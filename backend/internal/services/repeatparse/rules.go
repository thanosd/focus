// Package repeatparse turns phrases like "first of every month", "every
// other friday" or "every 2 weeks after completion" into a RepeatRule.
// The rule parser handles the everyday vocabulary; anything it rejects
// goes to the Claude fallback in ai.go.
package repeatparse

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thanosd/focus/backend/internal/domain"
)

// ErrNotUnderstood is returned when no parser could make sense of the phrase.
var ErrNotUnderstood = errors.New("could not understand repeat phrase")

// Result is a parsed repeat.
type Result struct {
	Rule domain.RepeatRule
	// Description is the plain-English reading of the rule.
	Description string
	// FirstOccurrence is set for anchored rules (weekdays / day of month /
	// an explicit date): the first date on or after now that matches, so a
	// task without a due date can be pinned to the calendar.
	FirstOccurrence *time.Time
	Source          string // "rules" | "ai"
}

// Request is one parse call.
type Request struct {
	Input    string
	Now      time.Time
	Location *time.Location
}

func (r Request) now() time.Time {
	if r.Now.IsZero() {
		r.Now = time.Now()
	}
	return r.Now.In(r.loc())
}

func (r Request) loc() *time.Location {
	if r.Location == nil {
		return time.UTC
	}
	return r.Location
}

var weekdayNames = map[string]time.Weekday{
	"sunday": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
	"wednesday": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday,
	"friday": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "sat": time.Saturday,
}

var monthNames = map[string]time.Month{
	"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4,
	"may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7, "aug": 8, "august": 8,
	"sep": 9, "sept": 9, "september": 9, "oct": 10, "october": 10, "nov": 11, "november": 11, "dec": 12, "december": 12,
}

var (
	reInterval   = regexp.MustCompile(`^(?:every|each)\s+(?:(\d+)|other|a|an)?\s*(day|days|week|weeks|month|months|year|years|wk|wks|mo|mos|yr|yrs)$`)
	reDayOfMonth = regexp.MustCompile(`^(?:(?:on\s+)?the\s+)?(\d{1,2})(?:st|nd|rd|th)?\s+(?:of\s+)?(?:every|each|the|a)\s+month$`)
	reMonthlyOn  = regexp.MustCompile(`^(?:monthly|every\s+month|each\s+month)\s+(?:on\s+)?(?:the\s+)?(\d{1,2})(?:st|nd|rd|th)?$`)
	reEveryNMon  = regexp.MustCompile(`^every\s+(\d+)\s+months?\s+(?:on\s+)?(?:the\s+)?(\d{1,2})(?:st|nd|rd|th)?$`)
	reYearlyOn   = regexp.MustCompile(`^(?:yearly|annually|every\s+year|each\s+year)\s+(?:on\s+)?([a-z]+)\.?\s+(\d{1,2})(?:st|nd|rd|th)?$`)
	reYearlyOn2  = regexp.MustCompile(`^(?:yearly|annually|every\s+year|each\s+year)\s+(?:on\s+)?(?:the\s+)?(\d{1,2})(?:st|nd|rd|th)?\s+(?:of\s+)?([a-z]+)\.?$`)
	reEveryDays  = regexp.MustCompile(`^(?:every|each|on)\s+(?:(\d+)\s+weeks?\s+on\s+|other\s+)?(.+?)$`)
)

// Parse resolves a phrase deterministically.
func Parse(req Request) (*Result, error) {
	in := normalize(req.Input)
	if in == "" {
		return nil, ErrNotUnderstood
	}
	from, in := splitFrom(in)
	now := req.now()

	rule := domain.RepeatRule{Every: 1, From: from}
	var first *time.Time
	ok := false

	switch in {
	case "daily", "every day", "each day", "once a day":
		rule.Unit, ok = domain.RepeatDay, true
	case "weekly", "every week", "each week", "once a week":
		rule.Unit, ok = domain.RepeatWeek, true
	case "biweekly", "fortnightly", "every two weeks", "every other week", "every 2nd week":
		rule.Unit, rule.Every, ok = domain.RepeatWeek, 2, true
	case "monthly", "every month", "each month", "once a month":
		rule.Unit, ok = domain.RepeatMonth, true
	case "quarterly", "every quarter", "every 3 months", "every three months":
		rule.Unit, rule.Every, ok = domain.RepeatMonth, 3, true
	case "yearly", "annually", "every year", "each year", "once a year":
		rule.Unit, ok = domain.RepeatYear, true
	case "every weekday", "weekdays", "each weekday", "every workday", "on weekdays":
		rule.Unit, rule.Weekdays, ok = domain.RepeatWeek, []time.Weekday{1, 2, 3, 4, 5}, true
	case "every weekend", "weekends", "on weekends":
		rule.Unit, rule.Weekdays, ok = domain.RepeatWeek, []time.Weekday{time.Saturday, time.Sunday}, true
	case "first of every month", "first of the month", "1st of every month", "1st of the month", "on the first of the month", "on the 1st of the month", "every first of the month", "start of every month", "beginning of every month", "start of the month":
		d := 1
		rule.Unit, rule.DayOfMonth, ok = domain.RepeatMonth, &d, true
	case "last day of every month", "last day of the month", "end of every month", "end of month", "end of the month", "on the last day of the month", "every end of month", "eom":
		d := -1
		rule.Unit, rule.DayOfMonth, ok = domain.RepeatMonth, &d, true
	}

	if !ok {
		if m := reInterval.FindStringSubmatch(in); m != nil {
			n := 1
			if m[1] != "" {
				n, _ = strconv.Atoi(m[1])
			} else if strings.HasPrefix(in, "every other") {
				n = 2
			}
			if n >= 1 {
				rule.Every, rule.Unit, ok = n, unitOf(m[2]), true
			}
		}
	}
	if !ok {
		for _, re := range []*regexp.Regexp{reDayOfMonth, reMonthlyOn} {
			if m := re.FindStringSubmatch(in); m != nil {
				if d, _ := strconv.Atoi(m[1]); d >= 1 && d <= 31 {
					rule.Unit, rule.DayOfMonth, ok = domain.RepeatMonth, &d, true
				}
				break
			}
		}
	}
	if !ok {
		if m := reEveryNMon.FindStringSubmatch(in); m != nil {
			n, _ := strconv.Atoi(m[1])
			if d, _ := strconv.Atoi(m[2]); n >= 1 && d >= 1 && d <= 31 {
				rule.Unit, rule.Every, rule.DayOfMonth, ok = domain.RepeatMonth, n, &d, true
			}
		}
	}
	if !ok {
		var mo time.Month
		var day int
		if m := reYearlyOn.FindStringSubmatch(in); m != nil {
			mo, day = monthNames[m[1]], atoi(m[2])
		} else if m := reYearlyOn2.FindStringSubmatch(in); m != nil {
			mo, day = monthNames[m[2]], atoi(m[1])
		}
		if mo != 0 && day >= 1 && day <= 31 {
			rule.Unit, ok = domain.RepeatYear, true
			f := time.Date(now.Year(), mo, day, 0, 0, 0, 0, req.loc())
			if !f.After(now) {
				f = f.AddDate(1, 0, 0)
			}
			first = &f
		}
	}
	if !ok {
		if m := reEveryDays.FindStringSubmatch(in); m != nil {
			days, dok := parseWeekdays(m[2])
			if dok {
				n := 1
				if m[1] != "" {
					n, _ = strconv.Atoi(m[1])
				} else if strings.Contains(in, " other ") {
					n = 2
				}
				rule.Unit, rule.Every, rule.Weekdays, ok = domain.RepeatWeek, n, days, true
			}
		}
	}
	if !ok {
		// Bare weekday list: "mondays and thursdays".
		if days, dok := parseWeekdays(in); dok {
			rule.Unit, rule.Weekdays, ok = domain.RepeatWeek, days, true
		}
	}
	if !ok {
		return nil, ErrNotUnderstood
	}
	if (rule.Anchored() || first != nil) && rule.From == "" {
		rule.From = domain.RepeatFromDue
	}
	if rule.From == "" {
		rule.From = domain.RepeatFromCompletion
	}
	if err := rule.Validate(); err != nil {
		return nil, ErrNotUnderstood
	}
	if first == nil && rule.Anchored() {
		// First calendar match on or after today (just before midnight so
		// Next can land on today). The interval doesn't matter for the first
		// one, only the anchor, so probe with Every = 1.
		probe := rule
		probe.Every = 1
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, req.loc()).Add(-time.Nanosecond)
		f := probe.Next(startOfToday)
		first = &f
	}
	return &Result{Rule: rule, Description: rule.Describe(), FirstOccurrence: first, Source: "rules"}, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func unitOf(s string) domain.RepeatUnit {
	switch {
	case strings.HasPrefix(s, "d"):
		return domain.RepeatDay
	case strings.HasPrefix(s, "w"):
		return domain.RepeatWeek
	case strings.HasPrefix(s, "m"):
		return domain.RepeatMonth
	default:
		return domain.RepeatYear
	}
}

// parseWeekdays reads "monday", "mon, wed and fri", "mondays", "tuesdays and thursdays".
func parseWeekdays(s string) ([]time.Weekday, bool) {
	s = strings.ReplaceAll(s, " and ", ",")
	s = strings.ReplaceAll(s, "&", ",")
	s = strings.ReplaceAll(s, "/", ",")
	var out []time.Weekday
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		p = strings.TrimSuffix(p, "s")
		if p == "" {
			continue
		}
		d, ok := weekdayNames[p]
		if !ok {
			return nil, false
		}
		out = append(out, d)
	}
	return out, len(out) > 0
}

// splitFrom peels a trailing "after completion" / "from due date" clause.
func splitFrom(in string) (domain.RepeatFrom, string) {
	for _, suf := range []string{" after completion", " after i finish", " after i complete it", " after completing", " from completion", " after done", " once done"} {
		if strings.HasSuffix(in, suf) {
			return domain.RepeatFromCompletion, strings.TrimSpace(strings.TrimSuffix(in, suf))
		}
	}
	for _, suf := range []string{" from due", " from due date", " from the due date", " regularly", " on schedule", " from defer", " from defer date"} {
		if strings.HasSuffix(in, suf) {
			return domain.RepeatFromDue, strings.TrimSpace(strings.TrimSuffix(in, suf))
		}
	}
	return "", in
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	for _, pre := range []string{"repeat ", "repeats ", "repeating ", "recur ", "recurs ", "recurring ", "do this ", "do it ", "remind me "} {
		s = strings.TrimPrefix(s, pre)
	}
	s = strings.TrimPrefix(s, "at ")
	s = strings.TrimPrefix(s, "on ")
	s = strings.TrimPrefix(s, "the ")
	return strings.TrimSpace(s)
}
