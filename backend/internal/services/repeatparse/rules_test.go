package repeatparse

import (
	"testing"
	"time"

	"github.com/thanosd/focus/backend/internal/domain"
)

func TestParse(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2026, 10, 1, 10, 0, 0, 0, loc) // Thursday
	one, fifteen, last := 1, 15, -1
	cases := []struct {
		in    string
		rule  domain.RepeatRule
		first string // yyyy-mm-dd of FirstOccurrence, "" if none
	}{
		{"every month", domain.RepeatRule{Every: 1, Unit: domain.RepeatMonth, From: domain.RepeatFromCompletion}, ""},
		{"monthly from due", domain.RepeatRule{Every: 1, Unit: domain.RepeatMonth, From: domain.RepeatFromDue}, ""},
		{"every 2 weeks", domain.RepeatRule{Every: 2, Unit: domain.RepeatWeek, From: domain.RepeatFromCompletion}, ""},
		{"every other week", domain.RepeatRule{Every: 2, Unit: domain.RepeatWeek, From: domain.RepeatFromCompletion}, ""},
		{"Repeat at the first of every month", domain.RepeatRule{Every: 1, Unit: domain.RepeatMonth, From: domain.RepeatFromDue, DayOfMonth: &one}, "2026-10-01"},
		{"on the 15th of each month", domain.RepeatRule{Every: 1, Unit: domain.RepeatMonth, From: domain.RepeatFromDue, DayOfMonth: &fifteen}, "2026-10-15"},
		{"monthly on the 15th after completion", domain.RepeatRule{Every: 1, Unit: domain.RepeatMonth, From: domain.RepeatFromCompletion, DayOfMonth: &fifteen}, "2026-10-15"},
		{"last day of the month", domain.RepeatRule{Every: 1, Unit: domain.RepeatMonth, From: domain.RepeatFromDue, DayOfMonth: &last}, "2026-10-31"},
		{"every 3 months on the 1st", domain.RepeatRule{Every: 3, Unit: domain.RepeatMonth, From: domain.RepeatFromDue, DayOfMonth: &one}, "2026-10-01"},
		{"every monday", domain.RepeatRule{Every: 1, Unit: domain.RepeatWeek, From: domain.RepeatFromDue, Weekdays: []time.Weekday{time.Monday}}, "2026-10-05"},
		{"mondays and thursdays", domain.RepeatRule{Every: 1, Unit: domain.RepeatWeek, From: domain.RepeatFromDue, Weekdays: []time.Weekday{time.Monday, time.Thursday}}, "2026-10-01"},
		{"every mon, wed & fri", domain.RepeatRule{Every: 1, Unit: domain.RepeatWeek, From: domain.RepeatFromDue, Weekdays: []time.Weekday{1, 3, 5}}, "2026-10-02"},
		{"every other friday", domain.RepeatRule{Every: 2, Unit: domain.RepeatWeek, From: domain.RepeatFromDue, Weekdays: []time.Weekday{time.Friday}}, "2026-10-02"},
		{"every weekday", domain.RepeatRule{Every: 1, Unit: domain.RepeatWeek, From: domain.RepeatFromDue, Weekdays: []time.Weekday{1, 2, 3, 4, 5}}, "2026-10-01"},
		{"daily", domain.RepeatRule{Every: 1, Unit: domain.RepeatDay, From: domain.RepeatFromCompletion}, ""},
		{"every 10 days", domain.RepeatRule{Every: 10, Unit: domain.RepeatDay, From: domain.RepeatFromCompletion}, ""},
		{"yearly on jan 1", domain.RepeatRule{Every: 1, Unit: domain.RepeatYear, From: domain.RepeatFromDue}, "2027-01-01"},
		{"every year on the 4th of july", domain.RepeatRule{Every: 1, Unit: domain.RepeatYear, From: domain.RepeatFromDue}, "2027-07-04"},
		{"quarterly", domain.RepeatRule{Every: 3, Unit: domain.RepeatMonth, From: domain.RepeatFromCompletion}, ""},
	}
	for _, c := range cases {
		got, err := Parse(Request{Input: c.in, Now: now, Location: loc})
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if got.Rule.Every != c.rule.Every || got.Rule.Unit != c.rule.Unit || got.Rule.From != c.rule.From ||
			!sameDays(got.Rule.Weekdays, c.rule.Weekdays) || !sameInt(got.Rule.DayOfMonth, c.rule.DayOfMonth) {
			t.Errorf("%q: got %+v want %+v", c.in, got.Rule, c.rule)
		}
		first := ""
		if got.FirstOccurrence != nil {
			first = got.FirstOccurrence.Format("2006-01-02")
		}
		if first != c.first {
			t.Errorf("%q: first occurrence %q want %q", c.in, first, c.first)
		}
		if got.Description == "" {
			t.Errorf("%q: empty description", c.in)
		}
	}
	for _, in := range []string{"", "whenever I feel like it", "every blue moon", "every 0 days"} {
		if _, err := Parse(Request{Input: in, Now: now}); err == nil {
			t.Errorf("%q: expected error", in)
		}
	}
}

func sameDays(a, b []time.Weekday) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
