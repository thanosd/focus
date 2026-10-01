package repeatparse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/thanosd/focus/backend/internal/domain"
)

// AIResolver is the fallback for phrases the rule parser rejects.
type AIResolver interface {
	Resolve(ctx context.Context, req Request) (*Result, error)
}

// ErrAIDisabled is returned when no Claude key is configured.
var ErrAIDisabled = errors.New("AI repeat parsing is not configured (CLAUDE_API_KEY is empty)")

// ClaudeResolver asks Claude to express the phrase as a RepeatRule.
type ClaudeResolver struct {
	client anthropic.Client
	model  string
}

// NewClaudeResolver returns nil, ErrAIDisabled when apiKey is empty.
func NewClaudeResolver(apiKey, model string) (*ClaudeResolver, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrAIDisabled
	}
	if model == "" {
		model = "claude-opus-5"
	}
	return &ClaudeResolver{client: anthropic.NewClient(option.WithAPIKey(apiKey)), model: model}, nil
}

const aiSystemPrompt = `You convert a short natural-language phrase from a to-do app into a repeat rule.
The rule model is:
  every: integer >= 1
  unit: "day" | "week" | "month" | "year"
  from: "due" (repeat on a fixed calendar schedule) | "completion" (next one is scheduled from when the task is finished)
  weekdays: optional list of integers 0=Sunday..6=Saturday, only with unit "week" ("every monday", "weekdays")
  day_of_month: optional integer 1-31 or -1 for the last day, only with unit "month" ("first of the month", "end of month")
  first_occurrence: optional "YYYY-MM-DD", the first date on/after today matching the schedule (set it for anything tied to the calendar, e.g. "every year on March 15")
Default "from" to "due" for calendar-anchored phrases and to "completion" when the phrase says "after I finish" or gives a bare interval.
Respond with ONLY a JSON object:
{"every": 1, "unit": "month", "from": "due", "weekdays": null, "day_of_month": 1, "first_occurrence": "2026-11-01", "description": "monthly on the 1st"}
If the phrase is not a repeat schedule at all, respond with {"error": "<why>"}.`

// Resolve asks Claude to interpret the phrase.
func (c *ClaudeResolver) Resolve(ctx context.Context, req Request) (*Result, error) {
	now := req.now()
	user := fmt.Sprintf("Today: %s\nTimezone: %s\nPhrase: %s", now.Format("Monday, 2006-01-02"), req.loc().String(), strings.TrimSpace(req.Input))
	resp, err := c.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:        anthropic.Model(c.model),
		MaxTokens:    300,
		Betas:        []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Fallbacks:    anthropic.BetaFallbacksParamOfDefault(),
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortLow},
		System:       []anthropic.BetaTextBlockParam{{Text: aiSystemPrompt}},
		Messages:     []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user))},
	})
	if err != nil {
		return nil, fmt.Errorf("claude repeat parse: %w", err)
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
	Every           int    `json:"every"`
	Unit            string `json:"unit"`
	From            string `json:"from"`
	Weekdays        []int  `json:"weekdays"`
	DayOfMonth      *int   `json:"day_of_month"`
	FirstOccurrence string `json:"first_occurrence"`
	Description     string `json:"description"`
	Error           string `json:"error"`
}

func parseAIResponse(raw string, req Request) (*Result, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("%w: malformed AI response", ErrNotUnderstood)
	}
	var parsed aiResponse
	if err := json.Unmarshal([]byte(raw[start:end+1]), &parsed); err != nil {
		return nil, fmt.Errorf("%w: malformed AI response", ErrNotUnderstood)
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrNotUnderstood, parsed.Error)
	}
	rule := domain.RepeatRule{Every: parsed.Every, Unit: domain.RepeatUnit(parsed.Unit), From: domain.RepeatFrom(parsed.From), DayOfMonth: parsed.DayOfMonth}
	for _, d := range parsed.Weekdays {
		rule.Weekdays = append(rule.Weekdays, time.Weekday(d))
	}
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("%w: AI returned an invalid rule: %v", ErrNotUnderstood, err)
	}
	res := &Result{Rule: rule, Description: rule.Describe(), Source: "ai"}
	if parsed.FirstOccurrence != "" {
		if f, err := time.ParseInLocation("2006-01-02", parsed.FirstOccurrence, req.loc()); err == nil {
			res.FirstOccurrence = &f
		}
	}
	return res, nil
}
