// Package domain holds the pure business types shared by every layer.
package domain

import (
	"errors"
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
type RepeatRule struct {
	Every int        `json:"every"`
	Unit  RepeatUnit `json:"unit"`
	From  RepeatFrom `json:"from"`
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
	return nil
}

// Add advances t by one repeat interval.
func (r RepeatRule) Add(t time.Time) time.Time {
	switch r.Unit {
	case RepeatDay:
		return t.AddDate(0, 0, r.Every)
	case RepeatWeek:
		return t.AddDate(0, 0, 7*r.Every)
	case RepeatMonth:
		return t.AddDate(0, r.Every, 0)
	case RepeatYear:
		return t.AddDate(r.Every, 0, 0)
	}
	return t
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
