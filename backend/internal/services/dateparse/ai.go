package dateparse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AIResolver is the fallback for phrases the rule parser rejects. The
// production implementation asks Claude; tests use a stub.
type AIResolver interface {
	Resolve(ctx context.Context, req Request) (*Result, error)
}

// ErrAIDisabled is returned when no Claude key is configured.
var ErrAIDisabled = errors.New("AI date parsing is not configured (CLAUDE_API_KEY is empty)")

// ClaudeResolver resolves phrases with one Messages call.
type ClaudeResolver struct {
	client anthropic.Client
	model  string
}

// NewClaudeResolver returns nil, ErrAIDisabled when apiKey is empty so the
// caller can run without the fallback instead of failing at startup.
func NewClaudeResolver(apiKey, model string) (*ClaudeResolver, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrAIDisabled
	}
	if model == "" {
		model = "claude-opus-5"
	}
	return &ClaudeResolver{client: anthropic.NewClient(option.WithAPIKey(apiKey)), model: model}, nil
}

const aiSystemPrompt = `You convert a short natural-language scheduling phrase from a to-do app into a concrete timestamp.
You are given the current date/time, the user's timezone, and whether the phrase is a DEFER date (when a task becomes available) or a DUE date.
Rules:
- Resolve relative phrases from the given "now". Prefer the nearest future interpretation.
- If the phrase names a date but no time: for a defer date use 00:00, for a due date use 17:00, in the user's timezone.
- "weekend" means Saturday; "next week" means the coming Monday; "end of month" means the last day of the month.
- Keep any explicit time the user wrote.
Respond with ONLY a JSON object, no prose:
{"datetime": "<RFC 3339 timestamp with the user's UTC offset>", "interpretation": "<short human description, e.g. 'Monday, Oct 5 at 9:00 AM'>"}
If the phrase is not a date or time at all, respond with {"error": "<why>"}.`

// Resolve asks Claude to interpret the phrase.
func (c *ClaudeResolver) Resolve(ctx context.Context, req Request) (*Result, error) {
	now := req.now()
	kind := req.Kind
	if kind == "" {
		kind = KindDefer
	}
	user := fmt.Sprintf("Now: %s (%s)\nTimezone: %s\nKind: %s\nPhrase: %s",
		now.Format("Monday, 2006-01-02 15:04 MST"), now.Format(time.RFC3339), req.loc().String(), kind, strings.TrimSpace(req.Input))

	// Server-side refusal fallbacks are on by default (per the Claude API
	// guidance for Opus 5 code); a safety decline on this harmless prompt
	// is vanishingly unlikely but costs nothing to cover.
	resp, err := c.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 300,
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Fallbacks: anthropic.BetaFallbacksParamOfDefault(),
		// Date arithmetic is routine: low effort keeps the round trip fast
		// while the user is waiting on the defer button.
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortLow},
		System:       []anthropic.BetaTextBlockParam{{Text: aiSystemPrompt}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude date parse: %w", err)
	}
	if resp.StopReason == anthropic.BetaStopReasonRefusal {
		return nil, fmt.Errorf("claude declined to parse the phrase")
	}
	var raw strings.Builder
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			raw.WriteString(tb.Text)
		}
	}
	return parseAIResponse(raw.String(), req)
}

type aiResponse struct {
	Datetime       string `json:"datetime"`
	Interpretation string `json:"interpretation"`
	Error          string `json:"error"`
}

func parseAIResponse(raw string, req Request) (*Result, error) {
	trimmed := extractJSON(raw)
	var parsed aiResponse
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, fmt.Errorf("%w: malformed AI response: %s", ErrNotUnderstood, strings.TrimSpace(raw))
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrNotUnderstood, parsed.Error)
	}
	at, err := time.Parse(time.RFC3339, parsed.Datetime)
	if err != nil {
		return nil, fmt.Errorf("%w: AI returned an invalid timestamp %q", ErrNotUnderstood, parsed.Datetime)
	}
	at = at.In(req.loc())
	label := strings.TrimSpace(parsed.Interpretation)
	if label == "" {
		label = at.Format("Mon, Jan 2 2006 at 3:04 PM MST")
	}
	return &Result{At: at, Interpretation: label, Source: SourceAI}, nil
}

func extractJSON(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < 0 || end < start {
		return strings.TrimSpace(raw)
	}
	return raw[start : end+1]
}
