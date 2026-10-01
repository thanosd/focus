package dateparse

import (
	"context"
	"errors"
	"log"
)

// Parser tries the rules first and falls back to the AI resolver.
type Parser struct {
	ai AIResolver // nil when the AI fallback is disabled
}

// NewParser constructs a Parser. ai may be nil.
func NewParser(ai AIResolver) *Parser {
	return &Parser{ai: ai}
}

// AIEnabled reports whether the fallback is wired.
func (p *Parser) AIEnabled() bool { return p != nil && p.ai != nil }

// Parse resolves a phrase. Errors wrap ErrNotUnderstood when neither
// parser could make sense of it.
func (p *Parser) Parse(ctx context.Context, req Request) (*Result, error) {
	res, err := ParseRules(req)
	if err == nil {
		return res, nil
	}
	if !errors.Is(err, ErrNotUnderstood) {
		return nil, err
	}
	if p == nil || p.ai == nil {
		return nil, err
	}
	aiRes, aiErr := p.ai.Resolve(ctx, req)
	if aiErr != nil {
		log.Printf("dateparse: AI fallback failed for %q: %v", req.Input, aiErr)
		if errors.Is(aiErr, ErrNotUnderstood) {
			return nil, aiErr
		}
		return nil, errors.Join(err, aiErr)
	}
	return aiRes, nil
}
