// Package domain holds the pure business types shared by every layer.
package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Sentinel errors mapped to HTTP statuses by the handlers.
var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
	ErrForbidden  = errors.New("forbidden")
)

// User is a Google-authenticated account.
type User struct {
	ID          string
	Email       string
	GoogleSub   string
	Name        string
	PictureURL  string
	Timezone    string
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Location returns the user's IANA timezone, falling back to UTC.
func (u User) Location() *time.Location {
	if u.Timezone != "" {
		if loc, err := time.LoadLocation(u.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// Session is a browser session referenced by the session cookie.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// OAuthState is the CSRF nonce for one Google sign-in round trip.
type OAuthState struct {
	State      string
	RedirectTo string
	ExpiresAt  time.Time
}

// APIToken authenticates MCP clients. Only the hash is persisted.
type APIToken struct {
	ID          string
	UserID      string
	Name        string
	TokenHash   string
	TokenPrefix string
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

// TaskStatus enumerates task lifecycle states.
type TaskStatus string

const (
	TaskActive    TaskStatus = "active"
	TaskCompleted TaskStatus = "completed"
	TaskDropped   TaskStatus = "dropped"
)

// ProjectStatus enumerates project lifecycle states.
type ProjectStatus string

const (
	ProjectActive    ProjectStatus = "active"
	ProjectOnHold    ProjectStatus = "on_hold"
	ProjectCompleted ProjectStatus = "completed"
	ProjectDropped   ProjectStatus = "dropped"
)

// ValidProjectStatus reports whether s is a known project status.
func ValidProjectStatus(s ProjectStatus) bool {
	switch s {
	case ProjectActive, ProjectOnHold, ProjectCompleted, ProjectDropped:
		return true
	}
	return false
}

// RepeatUnit is the unit of a repeat interval.
type RepeatUnit string

const (
	RepeatDay   RepeatUnit = "day"
	RepeatWeek  RepeatUnit = "week"
	RepeatMonth RepeatUnit = "month"
	RepeatYear  RepeatUnit = "year"
)

// RepeatFrom says what the next occurrence is scheduled relative to.
type RepeatFrom string

const (
	RepeatFromCompletion RepeatFrom = "completion"
	RepeatFromDue        RepeatFrom = "due"
)

// RepeatRule describes how a task repeats. Stored as JSONB on the task.
//
// Every/Unit is the interval. Two optional anchors refine it:
//   - Weekdays (unit=week): only these weekdays, e.g. Mon/Wed/Fri.
//   - DayOfMonth (unit=month): a fixed day, 1–31, or -1 for the last day;
//     days past the end of a month clamp to its last day.
//
// From says whether the next occurrence is computed from the completion
// time or from the previous due/defer date. Anchored rules are normally
// "from due" so they stay on their calendar day however late you finish.
type RepeatRule struct {
	Every      int            `json:"every"`
	Unit       RepeatUnit     `json:"unit"`
	From       RepeatFrom     `json:"from"`
	Weekdays   []time.Weekday `json:"weekdays,omitempty"`
	DayOfMonth *int           `json:"day_of_month,omitempty"`
}

// Validate normalises and checks a repeat rule.
func (r *RepeatRule) Validate() error {
	if r == nil {
		return nil
	}
	if r.Every < 1 {
		return errors.New("repeat_rule.every must be at least 1")
	}
	switch r.Unit {
	case RepeatDay, RepeatWeek, RepeatMonth, RepeatYear:
	default:
		return errors.New("repeat_rule.unit must be day, week, month or year")
	}
	if r.From == "" {
		r.From = RepeatFromCompletion
	}
	switch r.From {
	case RepeatFromCompletion, RepeatFromDue:
	default:
		return errors.New("repeat_rule.from must be completion or due")
	}
	if len(r.Weekdays) > 0 {
		if r.Unit != RepeatWeek {
			return errors.New("repeat_rule.weekdays only applies to weekly repeats")
		}
		seen := map[time.Weekday]bool{}
		clean := make([]time.Weekday, 0, len(r.Weekdays))
		for _, d := range r.Weekdays {
			if d < time.Sunday || d > time.Saturday {
				return errors.New("repeat_rule.weekdays must be 0 (Sunday) to 6 (Saturday)")
			}
			if !seen[d] {
				seen[d] = true
				clean = append(clean, d)
			}
		}
		sort.Slice(clean, func(i, j int) bool { return clean[i] < clean[j] })
		r.Weekdays = clean
	}
	if r.DayOfMonth != nil {
		if r.Unit != RepeatMonth {
			return errors.New("repeat_rule.day_of_month only applies to monthly repeats")
		}
		if d := *r.DayOfMonth; d == 0 || d < -1 || d > 31 {
			return errors.New("repeat_rule.day_of_month must be 1-31 or -1 for the last day")
		}
	}
	return nil
}

// Anchored reports whether the rule pins occurrences to calendar days.
func (r RepeatRule) Anchored() bool {
	return len(r.Weekdays) > 0 || r.DayOfMonth != nil
}

// Add advances t by one plain interval (ignores anchors).
func (r RepeatRule) Add(t time.Time) time.Time {
	switch r.Unit {
	case RepeatDay:
		return t.AddDate(0, 0, r.Every)
	case RepeatWeek:
		return t.AddDate(0, 0, 7*r.Every)
	case RepeatMonth:
		return addMonthsClamped(t, r.Every)
	case RepeatYear:
		return t.AddDate(r.Every, 0, 0)
	}
	return t
}

// Next returns the first occurrence strictly after t, honouring anchors
// and keeping t's time of day.
func (r RepeatRule) Next(t time.Time) time.Time {
	switch {
	case r.Unit == RepeatWeek && len(r.Weekdays) > 0:
		return r.nextWeekday(t)
	case r.Unit == RepeatMonth && r.DayOfMonth != nil:
		return r.nextDayOfMonth(t)
	default:
		return r.Add(t)
	}
}

// Upcoming lists the next n occurrences after t.
func (r RepeatRule) Upcoming(t time.Time, n int) []time.Time {
	out := make([]time.Time, 0, n)
	cur := t
	for i := 0; i < n; i++ {
		cur = r.Next(cur)
		out = append(out, cur)
	}
	return out
}

func (r RepeatRule) nextWeekday(t time.Time) time.Time {
	in := func(d time.Weekday) bool {
		for _, w := range r.Weekdays {
			if w == d {
				return true
			}
		}
		return false
	}
	// Rest of the current week (weeks start on Monday).
	for i := 1; i <= 7; i++ {
		c := t.AddDate(0, 0, i)
		if c.Weekday() == time.Monday && i > 0 && r.Every > 1 {
			break
		}
		if in(c.Weekday()) {
			return c
		}
	}
	// Jump to the start of the week `Every` weeks ahead and take its first match.
	daysToMonday := (int(t.Weekday()) + 6) % 7 // Monday=0 … Sunday=6
	weekStart := t.AddDate(0, 0, -daysToMonday)
	target := weekStart.AddDate(0, 0, 7*r.Every)
	for i := 0; i < 7; i++ {
		c := target.AddDate(0, 0, i)
		if in(c.Weekday()) {
			return c
		}
	}
	return r.Add(t)
}

func (r RepeatRule) nextDayOfMonth(t time.Time) time.Time {
	y, m := t.Year(), t.Month()
	for i := 0; i < 24; i++ {
		c := dayInMonth(y, m, *r.DayOfMonth, t)
		if c.After(t) {
			return c
		}
		// t is at or past this month's occurrence, so this month is the
		// current cycle: the next one is Every months on.
		y, m = addMonths(y, m, r.Every)
	}
	return r.Add(t)
}

func dayInMonth(y int, m time.Month, dom int, like time.Time) time.Time {
	last := time.Date(y, m+1, 0, 0, 0, 0, 0, like.Location()).Day()
	d := dom
	if dom == -1 || dom > last {
		d = last
	}
	return time.Date(y, m, d, like.Hour(), like.Minute(), like.Second(), 0, like.Location())
}

func addMonths(y int, m time.Month, n int) (int, time.Month) {
	total := int(m) - 1 + n
	return y + total/12, time.Month(total%12 + 1)
}

// addMonthsClamped is AddDate(0, n, 0) without the overflow (Jan 31 + 1
// month = Feb 28/29, not Mar 3).
func addMonthsClamped(t time.Time, n int) time.Time {
	y, m := addMonths(t.Year(), t.Month(), n)
	return dayInMonth(y, m, t.Day(), t)
}

// Describe renders the rule in plain English.
func (r RepeatRule) Describe() string {
	var b strings.Builder
	unit := string(r.Unit)
	switch {
	case r.Unit == RepeatWeek && len(r.Weekdays) > 0:
		names := make([]string, 0, len(r.Weekdays))
		for _, d := range r.Weekdays {
			names = append(names, d.String()[:3])
		}
		if r.Every == 1 {
			b.WriteString("every " + strings.Join(names, ", "))
		} else {
			fmt.Fprintf(&b, "every %d weeks on %s", r.Every, strings.Join(names, ", "))
		}
	case r.Unit == RepeatMonth && r.DayOfMonth != nil:
		day := "the last day"
		if *r.DayOfMonth > 0 {
			day = "the " + ordinal(*r.DayOfMonth)
		}
		if r.Every == 1 {
			b.WriteString("monthly on " + day)
		} else {
			fmt.Fprintf(&b, "every %d months on %s", r.Every, day)
		}
	case r.Every == 1:
		b.WriteString(map[RepeatUnit]string{RepeatDay: "daily", RepeatWeek: "weekly", RepeatMonth: "monthly", RepeatYear: "yearly"}[r.Unit])
	default:
		fmt.Fprintf(&b, "every %d %ss", r.Every, unit)
	}
	if r.From == RepeatFromCompletion {
		b.WriteString(" after completion")
	}
	return b.String()
}

func ordinal(n int) string {
	suffix := "th"
	switch {
	case n%100 >= 11 && n%100 <= 13:
	case n%10 == 1:
		suffix = "st"
	case n%10 == 2:
		suffix = "nd"
	case n%10 == 3:
		suffix = "rd"
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

// Tag is a user-defined label.
type Tag struct {
	ID              string
	UserID          string
	Name            string
	Color           string
	SortOrder       int
	ActiveTaskCount int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Task is a single action. ProjectID nil means it lives in the inbox.
type Task struct {
	ID          string
	UserID      string
	ProjectID   *string
	ProjectName string
	Title       string
	Note        string
	Status      TaskStatus
	Flagged     bool
	DeferUntil  *time.Time
	DueAt       *time.Time
	RepeatRule  *RepeatRule
	// IsAvailable is derived: active, not deferred into the future, and
	// either in the inbox or in an active project where sequential
	// ordering (if any) puts it first.
	IsAvailable bool
	Tags        []Tag
	SortOrder   int
	CompletedAt *time.Time
	DroppedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Project groups tasks. Depth is 0 (top level) or 1 (nested).
type Project struct {
	ID                 string
	UserID             string
	ParentID           *string
	Name               string
	Note               string
	Status             ProjectStatus
	Sequential         bool
	ReviewIntervalDays int
	LastReviewedAt     *time.Time
	NextReviewAt       *time.Time
	Depth              int
	SortOrder          int
	RemainingTaskCount int
	AvailableTaskCount int
	CompletedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TaskView names the canned task list filters.
type TaskView string

const (
	ViewInbox     TaskView = "inbox"
	ViewAvailable TaskView = "available"
	ViewFlagged   TaskView = "flagged"
	ViewDue       TaskView = "due"
	ViewCompleted TaskView = "completed"
	ViewAll       TaskView = "all"
)

// TaskFilter selects tasks for a list query.
type TaskFilter struct {
	View      TaskView
	ProjectID *string
	TagID     *string
	Query     string
}

// Counts are the sidebar badge numbers.
type Counts struct {
	Inbox     int
	Flagged   int
	DueSoon   int
	Overdue   int
	ReviewDue int
}
