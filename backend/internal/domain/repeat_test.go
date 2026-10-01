package domain

import (
	"testing"
	"time"
)

func date(y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, time.UTC)
}

func TestRepeatNext(t *testing.T) {
	one := 1
	fifteen := 15
	last := -1
	cases := []struct {
		name string
		rule RepeatRule
		from time.Time
		want time.Time
	}{
		{"daily", RepeatRule{Every: 1, Unit: RepeatDay}, date(2026, 10, 1, 9), date(2026, 10, 2, 9)},
		{"every 2 weeks", RepeatRule{Every: 2, Unit: RepeatWeek}, date(2026, 10, 1, 9), date(2026, 10, 15, 9)},
		{"monthly from Jan 31 clamps", RepeatRule{Every: 1, Unit: RepeatMonth}, date(2026, 1, 31, 9), date(2026, 2, 28, 9)},
		{"first of month, from mid month", RepeatRule{Every: 1, Unit: RepeatMonth, DayOfMonth: &one}, date(2026, 10, 15, 17), date(2026, 11, 1, 17)},
		{"first of month, from the 1st itself", RepeatRule{Every: 1, Unit: RepeatMonth, DayOfMonth: &one}, date(2026, 11, 1, 17), date(2026, 12, 1, 17)},
		{"15th every 3 months", RepeatRule{Every: 3, Unit: RepeatMonth, DayOfMonth: &fifteen}, date(2026, 10, 15, 9), date(2027, 1, 15, 9)},
		{"15th before it this month", RepeatRule{Every: 1, Unit: RepeatMonth, DayOfMonth: &fifteen}, date(2026, 10, 3, 9), date(2026, 10, 15, 9)},
		{"last day of month", RepeatRule{Every: 1, Unit: RepeatMonth, DayOfMonth: &last}, date(2026, 10, 31, 9), date(2026, 11, 30, 9)},
		{"last day from mid month", RepeatRule{Every: 1, Unit: RepeatMonth, DayOfMonth: &last}, date(2026, 2, 10, 9), date(2026, 2, 28, 9)},
		{"every monday from a wednesday", RepeatRule{Every: 1, Unit: RepeatWeek, Weekdays: []time.Weekday{time.Monday}}, date(2026, 9, 30, 9), date(2026, 10, 5, 9)},
		{"mon/wed/fri from a monday", RepeatRule{Every: 1, Unit: RepeatWeek, Weekdays: []time.Weekday{time.Monday, time.Wednesday, time.Friday}}, date(2026, 10, 5, 9), date(2026, 10, 7, 9)},
		{"weekdays from a friday", RepeatRule{Every: 1, Unit: RepeatWeek, Weekdays: []time.Weekday{1, 2, 3, 4, 5}}, date(2026, 10, 2, 9), date(2026, 10, 5, 9)},
		{"every other friday", RepeatRule{Every: 2, Unit: RepeatWeek, Weekdays: []time.Weekday{time.Friday}}, date(2026, 10, 2, 9), date(2026, 10, 16, 9)},
		{"yearly", RepeatRule{Every: 1, Unit: RepeatYear}, date(2026, 1, 1, 9), date(2027, 1, 1, 9)},
	}
	for _, c := range cases {
		if err := c.rule.Validate(); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got := c.rule.Next(c.from)
		if !got.Equal(c.want) {
			t.Errorf("%s: got %s want %s", c.name, got.Format(time.RFC3339), c.want.Format(time.RFC3339))
		}
	}
}

func TestRepeatUpcomingAndDescribe(t *testing.T) {
	one := 1
	r := RepeatRule{Every: 1, Unit: RepeatMonth, From: RepeatFromDue, DayOfMonth: &one}
	ups := r.Upcoming(date(2026, 10, 20, 17), 3)
	if len(ups) != 3 || ups[0].Day() != 1 || ups[0].Month() != time.November || ups[2].Month() != time.January {
		t.Fatalf("upcoming: %v", ups)
	}
	if got := r.Describe(); got != "monthly on the 1st" {
		t.Fatalf("describe: %q", got)
	}
	w := RepeatRule{Every: 2, Unit: RepeatWeek, From: RepeatFromCompletion, Weekdays: []time.Weekday{time.Monday, time.Friday}}
	if got := w.Describe(); got != "every 2 weeks on Mon, Fri after completion" {
		t.Fatalf("describe: %q", got)
	}
	if got := (RepeatRule{Every: 1, Unit: RepeatDay, From: RepeatFromCompletion}).Describe(); got != "daily after completion" {
		t.Fatalf("describe: %q", got)
	}
	bad := RepeatRule{Every: 1, Unit: RepeatDay, DayOfMonth: &one}
	if err := bad.Validate(); err == nil {
		t.Fatal("day_of_month on daily should be rejected")
	}
}
