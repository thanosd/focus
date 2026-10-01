package handlers

import (
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/thanosd/focus/backend/internal/api"
	"github.com/thanosd/focus/backend/internal/domain"
)

// Converters from domain types to the generated OpenAPI response types.
// Keeping them in one place means the wire shape is defined by the spec
// and the domain never leaks fields by accident.

func toAPIUser(u *domain.User) api.User {
	return api.User{
		Id:          uuidVal(u.ID),
		Email:       openapi_types.Email(u.Email),
		Name:        strPtr(u.Name),
		PictureUrl:  strPtr(u.PictureURL),
		Timezone:    u.Timezone,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}

func toAPITag(t domain.Tag) api.Tag {
	count := t.ActiveTaskCount
	return api.Tag{
		Id:              uuidVal(t.ID),
		Name:            t.Name,
		Color:           t.Color,
		ActiveTaskCount: &count,
		CreatedAt:       t.CreatedAt,
	}
}

func toAPITags(ts []domain.Tag) []api.Tag {
	out := make([]api.Tag, 0, len(ts))
	for _, t := range ts {
		out = append(out, toAPITag(t))
	}
	return out
}

func toAPIRepeat(r *domain.RepeatRule) *api.RepeatRule {
	if r == nil {
		return nil
	}
	return &api.RepeatRule{Every: r.Every, Unit: api.RepeatUnit(r.Unit), From: api.RepeatRuleFrom(r.From)}
}

func fromAPIRepeat(r *api.RepeatRule) *domain.RepeatRule {
	if r == nil {
		return nil
	}
	return &domain.RepeatRule{Every: r.Every, Unit: domain.RepeatUnit(r.Unit), From: domain.RepeatFrom(r.From)}
}

func toAPITask(t *domain.Task) api.Task {
	return api.Task{
		Id:          uuidVal(t.ID),
		ProjectId:   uuidPtr(t.ProjectID),
		ProjectName: strPtr(t.ProjectName),
		Title:       t.Title,
		Note:        t.Note,
		Status:      api.TaskStatus(t.Status),
		Flagged:     t.Flagged,
		DeferUntil:  t.DeferUntil,
		DueAt:       t.DueAt,
		RepeatRule:  toAPIRepeat(t.RepeatRule),
		IsAvailable: t.IsAvailable,
		Tags:        toAPITags(t.Tags),
		SortOrder:   t.SortOrder,
		CompletedAt: t.CompletedAt,
		DroppedAt:   t.DroppedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func toAPITasks(ts []domain.Task) []api.Task {
	out := make([]api.Task, 0, len(ts))
	for i := range ts {
		out = append(out, toAPITask(&ts[i]))
	}
	return out
}

func toAPIProject(p *domain.Project) api.Project {
	return api.Project{
		Id:                 uuidVal(p.ID),
		ParentId:           uuidPtr(p.ParentID),
		Name:               p.Name,
		Note:               p.Note,
		Status:             api.ProjectStatus(p.Status),
		Sequential:         p.Sequential,
		ReviewIntervalDays: p.ReviewIntervalDays,
		LastReviewedAt:     p.LastReviewedAt,
		NextReviewAt:       p.NextReviewAt,
		Depth:              p.Depth,
		SortOrder:          p.SortOrder,
		RemainingTaskCount: p.RemainingTaskCount,
		AvailableTaskCount: p.AvailableTaskCount,
		CompletedAt:        p.CompletedAt,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

func toAPIProjects(ps []domain.Project) []api.Project {
	out := make([]api.Project, 0, len(ps))
	for i := range ps {
		out = append(out, toAPIProject(&ps[i]))
	}
	return out
}

func toAPIToken(t domain.APIToken) api.ApiToken {
	return api.ApiToken{
		Id:          uuidVal(t.ID),
		Name:        t.Name,
		TokenPrefix: t.TokenPrefix,
		LastUsedAt:  t.LastUsedAt,
		CreatedAt:   t.CreatedAt,
	}
}
