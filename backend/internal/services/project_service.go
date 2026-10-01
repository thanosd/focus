package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
)

// ProjectService owns projects and reviews.
type ProjectService struct {
	projects ports.ProjectRepository
	tasks    ports.TaskRepository
}

// NewProjectService constructs a ProjectService.
func NewProjectService(projects ports.ProjectRepository, tasks ports.TaskRepository) *ProjectService {
	return &ProjectService{projects: projects, tasks: tasks}
}

// Get loads one project.
func (s *ProjectService) Get(ctx context.Context, userID, id string) (*domain.Project, error) {
	p, err := s.projects.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

// ProjectDetail bundles a project with its tasks and children.
type ProjectDetail struct {
	Project  *domain.Project
	Tasks    []domain.Task
	Children []domain.Project
}

// GetDetail loads a project with its active tasks and child projects.
func (s *ProjectService) GetDetail(ctx context.Context, userID, id string) (*ProjectDetail, error) {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	tasks, err := s.tasks.List(ctx, userID, domain.TaskFilter{View: domain.ViewAll, ProjectID: &id})
	if err != nil {
		return nil, err
	}
	children, err := s.projects.ListChildren(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return &ProjectDetail{Project: p, Tasks: tasks, Children: children}, nil
}

// List returns projects filtered by status ("" = active + on_hold, "all" = everything).
func (s *ProjectService) List(ctx context.Context, userID, status string) ([]domain.Project, error) {
	var statuses []domain.ProjectStatus
	switch status {
	case "", "open":
		statuses = []domain.ProjectStatus{domain.ProjectActive, domain.ProjectOnHold}
	case "all":
		statuses = []domain.ProjectStatus{domain.ProjectActive, domain.ProjectOnHold, domain.ProjectCompleted, domain.ProjectDropped}
	default:
		ps := domain.ProjectStatus(status)
		if !domain.ValidProjectStatus(ps) {
			return nil, fmt.Errorf("%w: unknown status %q", domain.ErrValidation, status)
		}
		statuses = []domain.ProjectStatus{ps}
	}
	return s.projects.List(ctx, userID, statuses)
}

// CreateProjectInput is the create payload.
type CreateProjectInput struct {
	Name               string
	Note               string
	ParentID           *string
	Sequential         bool
	ReviewIntervalDays int
}

// Create adds a project, enforcing the two-level nesting limit.
func (s *ProjectService) Create(ctx context.Context, userID string, in CreateProjectInput) (*domain.Project, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if in.ParentID != nil {
		if err := s.checkParent(ctx, userID, *in.ParentID, ""); err != nil {
			return nil, err
		}
	}
	if in.ReviewIntervalDays <= 0 {
		in.ReviewIntervalDays = 7
	}
	p := &domain.Project{
		UserID:             userID,
		ParentID:           in.ParentID,
		Name:               name,
		Note:               in.Note,
		Sequential:         in.Sequential,
		ReviewIntervalDays: in.ReviewIntervalDays,
	}
	if err := s.projects.Create(ctx, p); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, p.ID)
}

// checkParent verifies a prospective parent exists, is top-level, and
// (when re-parenting) isn't the project itself.
func (s *ProjectService) checkParent(ctx context.Context, userID, parentID, selfID string) error {
	if parentID == selfID {
		return fmt.Errorf("%w: a project cannot be its own parent", domain.ErrValidation)
	}
	parent, err := s.projects.GetByID(ctx, userID, parentID)
	if err != nil {
		return err
	}
	if parent == nil {
		return fmt.Errorf("%w: parent project not found", domain.ErrValidation)
	}
	if parent.ParentID != nil {
		return fmt.Errorf("%w: projects can only be nested two levels deep", domain.ErrValidation)
	}
	return nil
}

// ProjectPatch is a partial update. SetParent distinguishes "unset" from "null".
type ProjectPatch struct {
	Name               *string
	Note               *string
	SetParent          bool
	ParentID           *string
	Status             *domain.ProjectStatus
	Sequential         *bool
	ReviewIntervalDays *int
	SortOrder          *int
}

// Update applies a patch.
func (s *ProjectService) Update(ctx context.Context, userID, id string, p ProjectPatch) (*domain.Project, error) {
	proj, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
		}
		proj.Name = name
	}
	if p.Note != nil {
		proj.Note = *p.Note
	}
	if p.SetParent {
		if p.ParentID != nil {
			if err := s.checkParent(ctx, userID, *p.ParentID, id); err != nil {
				return nil, err
			}
			// A project with children cannot become a child itself.
			children, err := s.projects.ListChildren(ctx, userID, id)
			if err != nil {
				return nil, err
			}
			if len(children) > 0 {
				return nil, fmt.Errorf("%w: this project has sub-projects and cannot be nested", domain.ErrValidation)
			}
		}
		proj.ParentID = p.ParentID
	}
	if p.Status != nil {
		if !domain.ValidProjectStatus(*p.Status) {
			return nil, fmt.Errorf("%w: unknown status %q", domain.ErrValidation, *p.Status)
		}
		if *p.Status != proj.Status {
			proj.Status = *p.Status
			now := time.Now()
			if proj.Status == domain.ProjectCompleted || proj.Status == domain.ProjectDropped {
				proj.CompletedAt = &now
			} else {
				proj.CompletedAt = nil
			}
		}
	}
	if p.Sequential != nil {
		proj.Sequential = *p.Sequential
	}
	if p.ReviewIntervalDays != nil {
		if *p.ReviewIntervalDays < 1 {
			return nil, fmt.Errorf("%w: review_interval_days must be at least 1", domain.ErrValidation)
		}
		proj.ReviewIntervalDays = *p.ReviewIntervalDays
		base := time.Now()
		if proj.LastReviewedAt != nil {
			base = *proj.LastReviewedAt
		}
		next := base.AddDate(0, 0, proj.ReviewIntervalDays)
		proj.NextReviewAt = &next
	}
	if p.SortOrder != nil {
		proj.SortOrder = *p.SortOrder
	}
	if err := s.projects.Update(ctx, proj); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

// MarkReviewed stamps the review and schedules the next one.
func (s *ProjectService) MarkReviewed(ctx context.Context, userID, id string) (*domain.Project, error) {
	proj, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	next := now.AddDate(0, 0, proj.ReviewIntervalDays)
	proj.LastReviewedAt = &now
	proj.NextReviewAt = &next
	if err := s.projects.Update(ctx, proj); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

// Delete removes a project (children and tasks cascade).
func (s *ProjectService) Delete(ctx context.Context, userID, id string) error {
	return s.projects.Delete(ctx, userID, id)
}

// Reviews lists projects due and upcoming for review.
func (s *ProjectService) Reviews(ctx context.Context, userID string) (due, upcoming []domain.Project, err error) {
	now := time.Now()
	due, err = s.projects.ListReviewDue(ctx, userID, now)
	if err != nil {
		return nil, nil, err
	}
	upcoming, err = s.projects.ListReviewUpcoming(ctx, userID, now)
	if err != nil {
		return nil, nil, err
	}
	return due, upcoming, nil
}
