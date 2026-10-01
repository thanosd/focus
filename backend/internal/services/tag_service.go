package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
)

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// TagService owns tags.
type TagService struct {
	tags ports.TagRepository
}

// NewTagService constructs a TagService.
func NewTagService(tags ports.TagRepository) *TagService { return &TagService{tags: tags} }

// List returns all tags with active counts.
func (s *TagService) List(ctx context.Context, userID string) ([]domain.Tag, error) {
	return s.tags.List(ctx, userID)
}

// Get loads one tag.
func (s *TagService) Get(ctx context.Context, userID, id string) (*domain.Tag, error) {
	t, err := s.tags.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

// Create adds a tag; names are unique per user (case-insensitive).
func (s *TagService) Create(ctx context.Context, userID, name, color string) (*domain.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if color != "" && !hexColor.MatchString(color) {
		return nil, fmt.Errorf("%w: color must be a #rrggbb hex value", domain.ErrValidation)
	}
	if existing, err := s.tags.GetByName(ctx, userID, name); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("%w: a tag named %q already exists", domain.ErrValidation, existing.Name)
	}
	t := &domain.Tag{UserID: userID, Name: name, Color: color}
	if err := s.tags.Create(ctx, t); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, t.ID)
}

// EnsureByNames resolves tag names to IDs, creating missing ones. Used
// by the MCP server so Claude can say "tags: [errands, calls]".
func (s *TagService) EnsureByNames(ctx context.Context, userID string, names []string) ([]string, error) {
	ids := make([]string, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		t, err := s.tags.GetByName(ctx, userID, name)
		if err != nil {
			return nil, err
		}
		if t == nil {
			t = &domain.Tag{UserID: userID, Name: name}
			if err := s.tags.Create(ctx, t); err != nil {
				return nil, err
			}
		}
		ids = append(ids, t.ID)
	}
	return ids, nil
}

// TagPatch is a partial update.
type TagPatch struct {
	Name  *string
	Color *string
}

// Update applies a patch.
func (s *TagService) Update(ctx context.Context, userID, id string, p TagPatch) (*domain.Tag, error) {
	t, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
		}
		if other, err := s.tags.GetByName(ctx, userID, name); err != nil {
			return nil, err
		} else if other != nil && other.ID != id {
			return nil, fmt.Errorf("%w: a tag named %q already exists", domain.ErrValidation, other.Name)
		}
		t.Name = name
	}
	if p.Color != nil {
		if !hexColor.MatchString(*p.Color) {
			return nil, fmt.Errorf("%w: color must be a #rrggbb hex value", domain.ErrValidation)
		}
		t.Color = *p.Color
	}
	if err := s.tags.Update(ctx, t); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

// Delete removes a tag.
func (s *TagService) Delete(ctx context.Context, userID, id string) error {
	return s.tags.Delete(ctx, userID, id)
}
