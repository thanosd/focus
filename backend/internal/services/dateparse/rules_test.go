package dateparse

import (
	"testing"
	"time"
)

func TestParseRules(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	// Wednesday, 2026-09-30 10:30 local.
	now := time.Date(2026, 9, 30, 10, 30, 0, 0, loc)

	cases := []struct {
		in   string
		kind Kind
		want time.Time
	}{
		{"1d", KindDefer, time.Date(2026, 10, 1, 0, 0, 0, 0, loc)},
		{"+3d", KindDefer, time.Date(2026, 10, 3, 0, 0, 0, 0, loc)},
		{"1w", KindDefer, time.Date(2026, 10, 7, 0, 0, 0, 0, loc)},
		{"2 weeks", KindDue, time.Date(2026, 10, 14, 17, 0, 0, 0, loc)},
		{"1m", KindDefer, time.Date(2026, 10, 30, 0, 0, 0, 0, loc)},
		{"in 3 days", KindDefer, time.Date(2026, 10, 3, 0, 0, 0, 0, loc)},
		{"in 2 hours", KindDefer, time.Date(2026, 9, 30, 12, 30, 0, 0, loc)},
		{"tomorrow", KindDefer, time.Date(2026, 10, 1, 0, 0, 0, 0, loc)},
		{"Tomorrow at 3pm", KindDefer, time.Date(2026, 10, 1, 15, 0, 0, 0, loc)},
		{"today", KindDue, time.Date(2026, 9, 30, 17, 0, 0, 0, loc)},
		{"eod", KindDue, time.Date(2026, 9, 30, 17, 0, 0, 0, loc)},
		{"monday", KindDefer, time.Date(2026, 10, 5, 0, 0, 0, 0, loc)},
		{"next monday", KindDefer, time.Date(2026, 10, 5, 0, 0, 0, 0, loc)},
		{"wednesday", KindDefer, time.Date(2026, 10, 7, 0, 0, 0, 0, loc)},
		{"friday 9:30", KindDefer, time.Date(2026, 10, 2, 9, 30, 0, 0, loc)},
		{"next week", KindDefer, time.Date(2026, 10, 5, 0, 0, 0, 0, loc)},
		{"next month", KindDefer, time.Date(2026, 10, 1, 0, 0, 0, 0, loc)},
		{"eom", KindDefer, time.Date(2026, 9, 30, 0, 0, 0, 0, loc)},
		{"oct 5", KindDefer, time.Date(2026, 10, 5, 0, 0, 0, 0, loc)},
		{"October 5th at noon", KindDefer, time.Date(2026, 10, 5, 12, 0, 0, 0, loc)},
		{"5 nov", KindDue, time.Date(2026, 11, 5, 17, 0, 0, 0, loc)},
		{"jan 2", KindDefer, time.Date(2027, 1, 2, 0, 0, 0, 0, loc)},
		{"mid october", KindDefer, time.Date(2026, 10, 15, 0, 0, 0, 0, loc)},
		{"end of october", KindDefer, time.Date(2026, 10, 31, 0, 0, 0, 0, loc)},
		{"2026-12-24", KindDue, time.Date(2026, 12, 24, 17, 0, 0, 0, loc)},
		{"2026-12-24T08:15", KindDue, time.Date(2026, 12, 24, 8, 15, 0, 0, loc)},
		{"12/24", KindDefer, time.Date(2026, 12, 24, 0, 0, 0, 0, loc)},
		{"the 15th", KindDefer, time.Date(2026, 10, 15, 0, 0, 0, 0, loc)},
		{"3pm", KindDefer, time.Date(2026, 9, 30, 15, 0, 0, 0, loc)},
		{"9am", KindDefer, time.Date(2026, 10, 1, 9, 0, 0, 0, loc)},
		{"tonight", KindDefer, time.Date(2026, 9, 30, 18, 0, 0, 0, loc)},
		{"until next friday", KindDefer, time.Date(2026, 10, 2, 0, 0, 0, 0, loc)},
	}
	for _, c := range cases {
		got, err := ParseRules(Request{Input: c.in, Kind: c.kind, Now: now, Location: loc})
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.in, err)
			continue
		}
		if !got.At.Equal(c.want) {
			t.Errorf("%q: got %s want %s", c.in, got.At.Format(time.RFC3339), c.want.Format(time.RFC3339))
		}
		if got.Source != SourceRules || got.Interpretation == "" {
			t.Errorf("%q: bad result metadata %+v", c.in, got)
		}
	}
}

func TestParseRulesNotUnderstood(t *testing.T) {
	for _, in := range []string{"", "when the cows come home", "after the launch", "3 sleeps"} {
		if _, err := ParseRules(Request{Input: in}); err == nil {
			t.Errorf("%q: expected ErrNotUnderstood", in)
		}
	}
}
