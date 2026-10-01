package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
)

// TaskService owns tasks: inbox capture, editing, completion with
// repeats, deferral and the sidebar counts.
type TaskService struct {
	tasks    ports.TaskRepository
	projects ports.ProjectRepository
	parser   *dateparse.Parser
}

// NewTaskService constructs a TaskService.
func NewTaskService(tasks ports.TaskRepository, projects ports.ProjectRepository, parser *dateparse.Parser) *TaskService {
	return &TaskService{tasks: tasks, projects: projects, parser: parser}
}

// Get loads one task.
func (s *TaskService) Get(ctx context.Context, userID, id string) (*domain.Task, error) {
	t, err := s.tasks.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

// List applies a filter.
func (s *TaskService) List(ctx context.Context, userID string, f domain.TaskFilter) ([]domain.Task, error) {
	if f.View == "" && f.ProjectID == nil && f.TagID == nil && f.Query == "" {
		f.View = domain.ViewAvailable
	}
	if f.View == "" {
		f.View = domain.ViewAll
	}
	return s.tasks.List(ctx, userID, f)
}

// CreateTaskInput is the create payload.
type CreateTaskInput struct {
	Title      string
	Note       string
	ProjectID  *string
	Flagged    bool
	DeferUntil *time.Time
	DueAt      *time.Time
	RepeatRule *domain.RepeatRule
	TagIDs     []string
}

// Create adds a task (to the inbox when ProjectID is nil).
func (s *TaskService) Create(ctx context.Context, userID string, in CreateTaskInput) (*domain.Task, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrValidation)
	}
	if in.ProjectID != nil {
		if err := s.checkProject(ctx, userID, *in.ProjectID); err != nil {
			return nil, err
		}
	}
	if err := in.RepeatRule.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	t := &domain.Task{
		UserID:     userID,
		ProjectID:  in.ProjectID,
		Title:      title,
		Note:       in.Note,
		Flagged:    in.Flagged,
		DeferUntil: in.DeferUntil,
		DueAt:      in.DueAt,
		RepeatRule: in.RepeatRule,
	}
	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, err
	}
	if len(in.TagIDs) > 0 {
		if err := s.tasks.SetTags(ctx, userID, t.ID, in.TagIDs); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, userID, t.ID)
}

func (s *TaskService) checkProject(ctx context.Context, userID, projectID string) error {
	p, err := s.projects.GetByID(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("%w: project not found", domain.ErrValidation)
	}
	return nil
}

// TaskPatch is a partial update. The Set* flags distinguish "leave
// alone" from "clear" for nullable fields.
type TaskPatch struct {
	Title      *string
	Note       *string
	SetProject bool
	ProjectID  *string
	Flagged    *bool
	SetDefer   bool
	DeferUntil *time.Time
	SetDue     bool
	DueAt      *time.Time
	SetRepeat  bool
	RepeatRule *domain.RepeatRule
	TagIDs     []string // nil = unchanged
	SortOrder  *int
}

// Update applies a patch.
func (s *TaskService) Update(ctx context.Context, userID, id string, p TaskPatch) (*domain.Task, error) {
	t, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p.Title != nil {
		title := strings.TrimSpace(*p.Title)
		if title == "" {
			return nil, fmt.Errorf("%w: title is required", domain.ErrValidation)
		}
		t.Title = title
	}
	if p.Note != nil {
		t.Note = *p.Note
	}
	if p.SetProject {
		if p.ProjectID != nil {
			if err := s.checkProject(ctx, userID, *p.ProjectID); err != nil {
				return nil, err
			}
		}
		t.ProjectID = p.ProjectID
	}
	if p.Flagged != nil {
		t.Flagged = *p.Flagged
	}
	if p.SetDefer {
		t.DeferUntil = p.DeferUntil
	}
	if p.SetDue {
		t.DueAt = p.DueAt
	}
	if p.SetRepeat {
		if err := p.RepeatRule.Validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
		}
		t.RepeatRule = p.RepeatRule
	}
	if p.SortOrder != nil {
		t.SortOrder = *p.SortOrder
	}
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	if p.TagIDs != nil {
		if err := s.tasks.SetTags(ctx, userID, id, p.TagIDs); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, userID, id)
}

// Complete marks the task done. Repeating tasks spawn their next
// occurrence, which is returned as the second value.
func (s *TaskService) Complete(ctx context.Context, userID, id string) (*domain.Task, *domain.Task, error) {
	t, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	if t.Status == domain.TaskCompleted {
		return t, nil, nil
	}
	t.Status = domain.TaskCompleted
	t.CompletedAt = &now
	t.DroppedAt = nil
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, nil, err
	}

	var next *domain.Task
	if t.RepeatRule != nil {
		next, err = s.spawnNext(ctx, t, now)
		if err != nil {
			return nil, nil, err
		}
	}
	done, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	return done, next, nil
}

// spawnNext creates the next occurrence of a repeating task.
//
// from=completion: the defer date moves to completion + interval; if the
// task also had a due date the gap between defer and due is preserved.
// from=due: both dates advance by one interval from their previous values
// (so a weekly Friday task stays on Fridays no matter when it's done).
func (s *TaskService) spawnNext(ctx context.Context, done *domain.Task, completedAt time.Time) (*domain.Task, error) {
	rule := *done.RepeatRule
	next := &domain.Task{
		UserID:     done.UserID,
		ProjectID:  done.ProjectID,
		Title:      done.Title,
		Note:       done.Note,
		Flagged:    done.Flagged,
		RepeatRule: done.RepeatRule,
	}
	switch rule.From {
	case domain.RepeatFromDue:
		if done.DueAt != nil {
			d := rule.Add(*done.DueAt)
			// Catch up past-due repeats so the next one lands in the future.
			for !d.After(completedAt) {
				d = rule.Add(d)
			}
			next.DueAt = &d
			if done.DeferUntil != nil {
				gap := done.DueAt.Sub(*done.DeferUntil)
				df := d.Add(-gap)
				next.DeferUntil = &df
			}
		} else if done.DeferUntil != nil {
			d := rule.Add(*done.DeferUntil)
			for !d.After(completedAt) {
				d = rule.Add(d)
			}
			next.DeferUntil = &d
		} else {
			d := rule.Add(completedAt)
			next.DeferUntil = &d
		}
	default: // completion
		base := rule.Add(completedAt)
		if done.DeferUntil != nil {
			// Keep the original time of day for the defer date.
			y, m, d := base.Date()
			loc := done.DeferUntil.Location()
			base = time.Date(y, m, d, done.DeferUntil.In(loc).Hour(), done.DeferUntil.In(loc).Minute(), 0, 0, loc)
		}
		next.DeferUntil = &base
		if done.DueAt != nil {
			gap := time.Duration(0)
			if done.DeferUntil != nil {
				gap = done.DueAt.Sub(*done.DeferUntil)
			}
			due := base.Add(gap)
			if done.DeferUntil == nil {
				due = rule.Add(*done.DueAt)
				for !due.After(completedAt) {
					due = rule.Add(due)
				}
			}
			next.DueAt = &due
		}
	}
	if err := s.tasks.Create(ctx, next); err != nil {
		return nil, err
	}
	tagIDs := make([]string, 0, len(done.Tags))
	for _, g := range done.Tags {
		tagIDs = append(tagIDs, g.ID)
	}
	if len(tagIDs) > 0 {
		if err := s.tasks.SetTags(ctx, done.UserID, next.ID, tagIDs); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, done.UserID, next.ID)
}

// Reorder applies a drag-and-drop ordering. Every ID must be a task of
// the user; unknown IDs are rejected rather than silently ignored so the
// UI learns its list is stale.
func (s *TaskService) Reorder(ctx context.Context, userID string, taskIDs []string) ([]domain.Task, error) {
	if len(taskIDs) == 0 {
		return nil, fmt.Errorf("%w: task_ids is required", domain.ErrValidation)
	}
	seen := make(map[string]bool, len(taskIDs))
	for _, id := range taskIDs {
		if seen[id] {
			return nil, fmt.Errorf("%w: duplicate task id %s", domain.ErrValidation, id)
		}
		seen[id] = true
		if _, err := s.Get(ctx, userID, id); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("%w: task %s not found", domain.ErrValidation, id)
			}
			return nil, err
		}
	}
	if err := s.tasks.Reorder(ctx, userID, taskIDs); err != nil {
		return nil, err
	}
	out := make([]domain.Task, 0, len(taskIDs))
	for _, id := range taskIDs {
		t, err := s.Get(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, nil
}

// Drop abandons a task without completing it.
func (s *TaskService) Drop(ctx context.Context, userID, id string) (*domain.Task, error) {
	t, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	t.Status = domain.TaskDropped
	t.DroppedAt = &now
	t.CompletedAt = nil
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

// Reopen makes a completed/dropped task active again.
func (s *TaskService) Reopen(ctx context.Context, userID, id string) (*domain.Task, error) {
	t, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	t.Status = domain.TaskActive
	t.CompletedAt = nil
	t.DroppedAt = nil
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

// Delete removes a task permanently.
func (s *TaskService) Delete(ctx context.Context, userID, id string) error {
	return s.tasks.Delete(ctx, userID, id)
}

// DeferInput is one defer request: exactly one of Input/Until/Clear.
type DeferInput struct {
	Input    string
	Until    *time.Time
	Clear    bool
	Location *time.Location
}

// Defer sets the task's defer date from a phrase or timestamp.
func (s *TaskService) Defer(ctx context.Context, user *domain.User, id string, in DeferInput) (*domain.Task, *dateparse.Result, error) {
	t, err := s.Get(ctx, user.ID, id)
	if err != nil {
		return nil, nil, err
	}
	var parsed *dateparse.Result
	switch {
	case in.Clear:
		t.DeferUntil = nil
	case in.Until != nil:
		u := *in.Until
		t.DeferUntil = &u
	case strings.TrimSpace(in.Input) != "":
		loc := in.Location
		if loc == nil {
			loc = user.Location()
		}
		parsed, err = s.ParseDate(ctx, dateparse.Request{Input: in.Input, Kind: dateparse.KindDefer, Location: loc})
		if err != nil {
			return nil, nil, err
		}
		at := parsed.At
		t.DeferUntil = &at
	default:
		return nil, nil, fmt.Errorf("%w: provide input, until or clear", domain.ErrValidation)
	}
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, nil, err
	}
	t, err = s.Get(ctx, user.ID, id)
	return t, parsed, err
}

// ParseDate resolves a phrase, mapping parser failures to validation errors.
func (s *TaskService) ParseDate(ctx context.Context, req dateparse.Request) (*dateparse.Result, error) {
	res, err := s.parser.Parse(ctx, req)
	if err != nil {
		if errors.Is(err, dateparse.ErrNotUnderstood) {
			return nil, fmt.Errorf("%w: couldn't understand %q", domain.ErrValidation, strings.TrimSpace(req.Input))
		}
		return nil, err
	}
	return res, nil
}

// Counts returns the sidebar badges.
func (s *TaskService) Counts(ctx context.Context, userID string) (*domain.Counts, error) {
	return s.tasks.Counts(ctx, userID, time.Now())
}
