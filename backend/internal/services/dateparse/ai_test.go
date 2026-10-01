package dateparse

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubAI struct {
	res *Result
	err error
}

func (s stubAI) Resolve(context.Context, Request) (*Result, error) { return s.res, s.err }

func TestParserFallsBackToAI(t *testing.T) {
	want := time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC)
	p := NewParser(stubAI{res: &Result{At: want, Interpretation: "after the launch", Source: SourceAI}})
	got, err := p.Parse(context.Background(), Request{Input: "after the launch", Now: want.AddDate(0, 0, -5)})
	if err != nil {
		t.Fatal(err)
	}
	if !got.At.Equal(want) || got.Source != SourceAI {
		t.Fatalf("unexpected result %+v", got)
	}
}

func TestParserRulesWinWithoutAI(t *testing.T) {
	p := NewParser(stubAI{err: errors.New("should not be called")})
	if _, err := p.Parse(context.Background(), Request{Input: "tomorrow", Now: time.Now()}); err != nil {
		t.Fatal(err)
	}
}

func TestParserNoAI(t *testing.T) {
	p := NewParser(nil)
	_, err := p.Parse(context.Background(), Request{Input: "whenever"})
	if !errors.Is(err, ErrNotUnderstood) {
		t.Fatalf("expected ErrNotUnderstood, got %v", err)
	}
}

func TestParseAIResponse(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	req := Request{Location: loc}
	res, err := parseAIResponse("Sure:\n{\"datetime\":\"2026-10-05T09:00:00-04:00\",\"interpretation\":\"Monday morning\"}", req)
	if err != nil {
		t.Fatal(err)
	}
	if res.At.Hour() != 9 || res.Interpretation != "Monday morning" {
		t.Fatalf("unexpected %+v", res)
	}
	if _, err := parseAIResponse(`{"error":"not a date"}`, req); !errors.Is(err, ErrNotUnderstood) {
		t.Fatalf("expected ErrNotUnderstood, got %v", err)
	}
	if _, err := parseAIResponse(`garbage`, req); !errors.Is(err, ErrNotUnderstood) {
		t.Fatalf("expected ErrNotUnderstood, got %v", err)
	}
}
