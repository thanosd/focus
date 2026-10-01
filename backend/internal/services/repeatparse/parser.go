package repeatparse

import (
	"context"
	"errors"
	"log"
)

// Parser tries the rules first and falls back to the AI resolver.
type Parser struct{ ai AIResolver }

// NewParser constructs a Parser; ai may be nil.
func NewParser(ai AIResolver) *Parser { return &Parser{ai: ai} }

// Parse resolves a phrase. Errors wrap ErrNotUnderstood when neither
// parser could make sense of it.
func (p *Parser) Parse(ctx context.Context, req Request) (*Result, error) {
	res, err := Parse(req)
	if err == nil {
		return res, nil
	}
	if p == nil || p.ai == nil {
		return nil, err
	}
	aiRes, aiErr := p.ai.Resolve(ctx, req)
	if aiErr != nil {
		log.Printf("repeatparse: AI fallback failed for %q: %v", req.Input, aiErr)
		if errors.Is(aiErr, ErrNotUnderstood) {
			return nil, aiErr
		}
		return nil, errors.Join(err, aiErr)
	}
	return aiRes, nil
}
