// Package dateparse resolves natural-language phrases ("1w", "next
// monday", "in 3 days", "mid october at 3pm") to timestamps.
//
// The deterministic parser in this file handles the phrases people type
// dozens of times a day; anything it can't understand is handed to the
// AI fallback in ai.go. Both produce a Result with the same shape so
// callers don't care which path answered.
package dateparse

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Kind selects the default time of day when a phrase names only a date.
type Kind string

const (
	// KindDefer resolves to the start of the day (00:00 local).
	KindDefer Kind = "defer"
	// KindDue resolves to 17:00 local.
	KindDue Kind = "due"
)

// Source says which parser produced the result.
type Source string

const (
	SourceRules Source = "rules"
	SourceAI    Source = "ai"
)

// Result is a resolved phrase.
type Result struct {
	At             time.Time
	Interpretation string
	Source         Source
}

// ErrNotUnderstood is returned by the rule parser when it has no match.
var ErrNotUnderstood = errors.New("could not understand date phrase")

// Request is one parse call.
type Request struct {
	Input    string
	Kind     Kind
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

// timeOfDay is an optional explicit clock time parsed off the end of a phrase.
type timeOfDay struct {
	hour, minute int
	set          bool
}

var (
	reRelative = regexp.MustCompile(`^(?:in\s+|\+)?\s*(\d+)\s*(d|day|days|w|wk|wks|week|weeks|m|mo|mos|month|months|y|yr|yrs|year|years|h|hr|hrs|hour|hours|min|mins|minute|minutes)$`)
	reClock    = regexp.MustCompile(`(?:^|\s+)(?:at\s+|@\s*)?(\d{1,2})(?::(\d{2}))?\s*(am|pm|a|p)?$`)
	reISO      = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})(?:[t ](\d{2}):(\d{2}))?$`)
	reUS       = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})(?:/(\d{2,4}))?$`)
	reMonthDay = regexp.MustCompile(`^([a-z]+)\.?\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?$`)
	reDayMonth = regexp.MustCompile(`^(\d{1,2})(?:st|nd|rd|th)?\s+([a-z]+)\.?(?:,?\s+(\d{4}))?$`)
	reOrdinal  = regexp.MustCompile(`^(?:the\s+)?(\d{1,2})(?:st|nd|rd|th)$`)
	reMonthPos = regexp.MustCompile(`^(early|mid|mid-|middle of|late|end of|eo)\s*([a-z]+)$`)
)

var weekdays = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "weds": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

var months = map[string]time.Month{
	"jan": time.January, "january": time.January,
	"feb": time.February, "february": time.February,
	"mar": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"may": time.May,
	"jun": time.June, "june": time.June,
	"jul": time.July, "july": time.July,
	"aug": time.August, "august": time.August,
	"sep": time.September, "sept": time.September, "september": time.September,
	"oct": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"dec": time.December, "december": time.December,
}

// ParseRules resolves a phrase deterministically. It returns
// ErrNotUnderstood when the phrase doesn't match any known shape.
func ParseRules(req Request) (*Result, error) {
	input := normalize(req.Input)
	if input == "" {
		return nil, ErrNotUnderstood
	}
	now := req.now()
	loc := req.loc()

	// Exact hours/minutes are "from now" and keep their clock time.
	if m := reRelative.FindStringSubmatch(input); m != nil {
		n, _ := strconv.Atoi(m[1])
		unit := m[2]
		switch {
		case strings.HasPrefix(unit, "h"):
			at := now.Add(time.Duration(n) * time.Hour)
			return result(at, fmt.Sprintf("in %d hour%s", n, plural(n))), nil
		case strings.HasPrefix(unit, "min"):
			at := now.Add(time.Duration(n) * time.Minute)
			return result(at, fmt.Sprintf("in %d minute%s", n, plural(n))), nil
		}
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		var day time.Time
		var label string
		switch unit[0] {
		case 'd':
			day, label = today.AddDate(0, 0, n), fmt.Sprintf("in %d day%s", n, plural(n))
		case 'w':
			day, label = today.AddDate(0, 0, 7*n), fmt.Sprintf("in %d week%s", n, plural(n))
		case 'm':
			day, label = today.AddDate(0, n, 0), fmt.Sprintf("in %d month%s", n, plural(n))
		case 'y':
			day, label = today.AddDate(n, 0, 0), fmt.Sprintf("in %d year%s", n, plural(n))
		}
		return result(withDefaultTime(day, req.Kind, timeOfDay{}), label), nil
	}

	// Split off a trailing clock time ("next friday at 3pm", "tomorrow 9:30").
	phrase, tod := splitClock(input)
	if phrase == "" {
		// Only a time was given: today at that time, or tomorrow if passed.
		at := time.Date(now.Year(), now.Month(), now.Day(), tod.hour, tod.minute, 0, 0, loc)
		if !at.After(now) {
			at = at.AddDate(0, 0, 1)
		}
		return result(at, "at "+clockLabel(tod)), nil
	}

	if day, label, ok := parseDayPhrase(phrase, now, loc, req.Kind); ok {
		return result(withDefaultTime(day, req.Kind, tod), label), nil
	}

	return nil, ErrNotUnderstood
}

// parseDayPhrase resolves the date part of a phrase to midnight (local)
// on the target day. Phrases that carry their own time (eod, noon…)
// are handled by splitClock / withDefaultTime instead.
func parseDayPhrase(p string, now time.Time, loc *time.Location, kind Kind) (time.Time, string, bool) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	switch p {
	case "today", "now", "tod":
		return today, "today", true
	case "tomorrow", "tmrw", "tmr", "tom":
		return today.AddDate(0, 0, 1), "tomorrow", true
	case "day after tomorrow":
		return today.AddDate(0, 0, 2), "the day after tomorrow", true
	case "next week":
		return nextWeekday(today, time.Monday, true), "next week (Monday)", true
	case "this weekend", "weekend", "saturday", "sat":
		return nextWeekday(today, time.Saturday, false), "Saturday", true
	case "next weekend":
		return nextWeekday(today, time.Saturday, true), "next weekend (Saturday)", true
	case "next month":
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, loc), "the 1st of next month", true
	case "end of month", "eom", "end of the month":
		return time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, loc), "the end of this month", true
	case "end of week", "eow", "end of the week":
		return nextWeekday(today, time.Friday, false), "Friday", true
	case "next year":
		return time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, loc), "January 1 next year", true
	case "end of year", "eoy":
		return time.Date(now.Year(), 12, 31, 0, 0, 0, 0, loc), "December 31", true
	}

	// "monday", "next monday", "this friday"
	rest := p
	strict := false
	if strings.HasPrefix(rest, "next ") {
		rest = strings.TrimPrefix(rest, "next ")
		strict = true
	} else if strings.HasPrefix(rest, "this ") {
		rest = strings.TrimPrefix(rest, "this ")
	}
	if wd, ok := weekdays[rest]; ok {
		day := nextWeekday(today, wd, strict)
		label := day.Weekday().String()
		if strict {
			label = "next " + label
		}
		return day, label, true
	}
	if mo, ok := months[rest]; ok {
		day := nextMonth(today, mo, 1)
		return day, mo.String() + " 1", true
	}

	if m := reISO.FindStringSubmatch(p); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		day := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, loc)
		if m[4] != "" {
			h, _ := strconv.Atoi(m[4])
			mi, _ := strconv.Atoi(m[5])
			day = time.Date(y, time.Month(mo), d, h, mi, 0, 0, loc)
		}
		return day, day.Format("Monday, January 2, 2006"), true
	}
	if m := reUS.FindStringSubmatch(p); m != nil {
		mo, _ := strconv.Atoi(m[1])
		d, _ := strconv.Atoi(m[2])
		if mo < 1 || mo > 12 || d < 1 || d > 31 {
			return time.Time{}, "", false
		}
		var day time.Time
		if m[3] != "" {
			y, _ := strconv.Atoi(m[3])
			if y < 100 {
				y += 2000
			}
			day = time.Date(y, time.Month(mo), d, 0, 0, 0, 0, loc)
		} else {
			day = nextMonth(today, time.Month(mo), d)
		}
		return day, day.Format("Monday, January 2, 2006"), true
	}
	if m := reMonthDay.FindStringSubmatch(p); m != nil {
		if mo, ok := months[m[1]]; ok {
			return monthDayYear(today, loc, mo, m[2], m[3])
		}
	}
	if m := reDayMonth.FindStringSubmatch(p); m != nil {
		if mo, ok := months[m[2]]; ok {
			return monthDayYear(today, loc, mo, m[1], m[3])
		}
	}
	if m := reOrdinal.FindStringSubmatch(p); m != nil {
		d, _ := strconv.Atoi(m[1])
		if d >= 1 && d <= 31 {
			day := time.Date(now.Year(), now.Month(), d, 0, 0, 0, 0, loc)
			if !day.After(today) {
				day = time.Date(now.Year(), now.Month()+1, d, 0, 0, 0, 0, loc)
			}
			return day, day.Format("Monday, January 2"), true
		}
	}
	if m := reMonthPos.FindStringSubmatch(p); m != nil {
		if mo, ok := months[m[2]]; ok {
			var d int
			var label string
			switch strings.TrimSpace(m[1]) {
			case "early":
				d, label = 1, "early "+mo.String()
			case "mid", "mid-", "middle of":
				d, label = 15, "mid "+mo.String()
			case "late":
				d, label = 25, "late "+mo.String()
			default: // end of / eo
				d, label = 0, "the end of "+mo.String()
			}
			day := nextMonth(today, mo, 1)
			if d == 0 {
				day = time.Date(day.Year(), day.Month()+1, 0, 0, 0, 0, 0, loc)
			} else {
				day = time.Date(day.Year(), day.Month(), d, 0, 0, 0, 0, loc)
			}
			return day, label, true
		}
	}
	return time.Time{}, "", false
}

func monthDayYear(today time.Time, loc *time.Location, mo time.Month, dayStr, yearStr string) (time.Time, string, bool) {
	d, _ := strconv.Atoi(dayStr)
	if d < 1 || d > 31 {
		return time.Time{}, "", false
	}
	var day time.Time
	if yearStr != "" {
		y, _ := strconv.Atoi(yearStr)
		day = time.Date(y, mo, d, 0, 0, 0, 0, loc)
	} else {
		day = nextMonth(today, mo, d)
	}
	return day, day.Format("Monday, January 2, 2006"), true
}

// nextWeekday returns the next occurrence of wd strictly after today.
// With strict=true ("next monday") and today being that weekday it
// still returns a week ahead; without strict it does too — "monday" on
// a Monday means next week's Monday in everyday usage.
func nextWeekday(today time.Time, wd time.Weekday, strict bool) time.Time {
	delta := (int(wd) - int(today.Weekday()) + 7) % 7
	if delta == 0 {
		delta = 7
	}
	_ = strict // "next monday" and "monday" resolve the same way; kept for callers' readability
	return today.AddDate(0, 0, delta)
}

// nextMonth returns day d of month mo, in this year if that's still in
// the future, otherwise next year. Day 1 is used when d is out of range.
func nextMonth(today time.Time, mo time.Month, d int) time.Time {
	if d < 1 || d > 31 {
		d = 1
	}
	candidate := time.Date(today.Year(), mo, d, 0, 0, 0, 0, today.Location())
	if !candidate.After(today) {
		candidate = time.Date(today.Year()+1, mo, d, 0, 0, 0, 0, today.Location())
	}
	return candidate
}

// withDefaultTime applies an explicit clock time, or the kind's default.
func withDefaultTime(day time.Time, kind Kind, tod timeOfDay) time.Time {
	loc := day.Location()
	if tod.set {
		return time.Date(day.Year(), day.Month(), day.Day(), tod.hour, tod.minute, 0, 0, loc)
	}
	if day.Hour() != 0 || day.Minute() != 0 {
		// Phrase carried its own time (ISO with time, "in 3 hours").
		return day
	}
	if kind == KindDue {
		return time.Date(day.Year(), day.Month(), day.Day(), 17, 0, 0, 0, loc)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
}

// splitClock peels a trailing time expression off the phrase.
func splitClock(input string) (string, timeOfDay) {
	named := map[string]timeOfDay{
		"noon": {12, 0, true}, "midday": {12, 0, true}, "midnight": {0, 0, true},
		"morning": {9, 0, true}, "afternoon": {14, 0, true}, "evening": {18, 0, true},
		"tonight": {18, 0, true}, "eod": {17, 0, true}, "end of day": {17, 0, true},
		"cob": {17, 0, true}, "lunch": {12, 0, true}, "lunchtime": {12, 0, true},
	}
	for name, tod := range named {
		switch {
		case input == name:
			return "today", tod
		case strings.HasSuffix(input, " at "+name):
			return strings.TrimSpace(strings.TrimSuffix(input, " at "+name)), tod
		case strings.HasSuffix(input, " in the "+name):
			return strings.TrimSpace(strings.TrimSuffix(input, " in the "+name)), tod
		case strings.HasSuffix(input, " "+name):
			return strings.TrimSpace(strings.TrimSuffix(input, " "+name)), tod
		}
	}
	m := reClock.FindStringSubmatchIndex(input)
	if m == nil {
		return input, timeOfDay{}
	}
	hourStr := input[m[2]:m[3]]
	hour, _ := strconv.Atoi(hourStr)
	minute := 0
	if m[4] >= 0 {
		minute, _ = strconv.Atoi(input[m[4]:m[5]])
	}
	ampm := ""
	if m[6] >= 0 {
		ampm = input[m[6]:m[7]]
	}
	// A bare number without am/pm or minutes is ambiguous ("oct 5",
	// "the 15th") — only treat it as a time when it has a colon or am/pm.
	if ampm == "" && m[4] < 0 {
		return input, timeOfDay{}
	}
	if hour > 23 || minute > 59 {
		return input, timeOfDay{}
	}
	if (ampm == "pm" || ampm == "p") && hour < 12 {
		hour += 12
	}
	if (ampm == "am" || ampm == "a") && hour == 12 {
		hour = 0
	}
	rest := strings.TrimSpace(input[:m[0]])
	return rest, timeOfDay{hour: hour, minute: minute, set: true}
}

func clockLabel(t timeOfDay) string {
	return time.Date(2000, 1, 1, t.hour, t.minute, 0, 0, time.UTC).Format("3:04 PM")
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimPrefix(s, "on ")
	s = strings.TrimPrefix(s, "by ")
	s = strings.TrimPrefix(s, "until ")
	s = strings.TrimPrefix(s, "defer to ")
	s = strings.TrimPrefix(s, "defer until ")
	s = strings.TrimPrefix(s, "due ")
	return strings.TrimSpace(s)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func result(at time.Time, label string) *Result {
	return &Result{
		At:             at,
		Interpretation: fmt.Sprintf("%s — %s", label, at.Format("Mon, Jan 2 2006 at 3:04 PM MST")),
		Source:         SourceRules,
	}
}
