package adapters_test

import (
	"context"
	"testing"
	"time"

	"github.com/thanosd/focus/backend/internal/adapters"
	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/services"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
	"github.com/thanosd/focus/backend/internal/testdb"
)

func TestRepositoriesAndServices(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()

	users := adapters.NewUserRepository(db)
	sessions := adapters.NewSessionRepository(db)
	states := adapters.NewOAuthStateRepository(db)
	tokens := adapters.NewAPITokenRepository(db)
	projects := adapters.NewProjectRepository(db)
	tasks := adapters.NewTaskRepository(db)
	tags := adapters.NewTagRepository(db)

	user, err := users.UpsertFromGoogle(ctx, "Thanos@Example.com", "sub-1", "Thanos", "https://pic")
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	if user.Email != "thanos@example.com" || user.Timezone != "UTC" {
		t.Fatalf("unexpected user %+v", user)
	}
	again, err := users.UpsertFromGoogle(ctx, "thanos@example.com", "sub-1", "", "")
	if err != nil || again.ID != user.ID || again.Name != "Thanos" {
		t.Fatalf("second upsert should keep the row: %+v %v", again, err)
	}
	if err := users.UpdateTimezone(ctx, user.ID, "America/New_York"); err != nil {
		t.Fatal(err)
	}

	// Sessions + oauth states + tokens
	sess := &domain.Session{UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}
	if err := sessions.Create(ctx, sess); err != nil {
		t.Fatal(err)
	}
	if got, _ := sessions.GetValid(ctx, sess.ID); got == nil {
		t.Fatal("session should be valid")
	}
	expired := &domain.Session{UserID: user.ID, ExpiresAt: time.Now().Add(-time.Hour)}
	_ = sessions.Create(ctx, expired)
	if n, _ := sessions.DeleteExpired(ctx); n != 1 {
		t.Fatalf("expected 1 expired session deleted, got %d", n)
	}
	_ = states.Create(ctx, &domain.OAuthState{State: "abc", RedirectTo: "/inbox", ExpiresAt: time.Now().Add(time.Minute)})
	if st, _ := states.Consume(ctx, "abc"); st == nil || st.RedirectTo != "/inbox" {
		t.Fatal("state should be consumable once")
	}
	if st, _ := states.Consume(ctx, "abc"); st != nil {
		t.Fatal("state should be gone")
	}

	cfgAuth, err := services.NewAuthService(ctx, testConfig(), users, sessions, states, tokens)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, tok, err := cfgAuth.CreateAPIToken(ctx, user.ID, "laptop")
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := cfgAuth.UserFromAPIToken(ctx, plaintext); u == nil || u.ID != user.ID {
		t.Fatal("token should resolve to user")
	}
	if u, _ := cfgAuth.UserFromAPIToken(ctx, "fcs_nope"); u != nil {
		t.Fatal("bad token should not resolve")
	}
	if err := cfgAuth.RevokeAPIToken(ctx, user.ID, tok.ID); err != nil {
		t.Fatal(err)
	}
	if u, _ := cfgAuth.UserFromAPIToken(ctx, plaintext); u != nil {
		t.Fatal("revoked token should not resolve")
	}

	// Projects: two-level nesting
	parser := dateparse.NewParser(nil)
	taskSvc := services.NewTaskService(tasks, projects, parser)
	projSvc := services.NewProjectService(projects, tasks)
	tagSvc := services.NewTagService(tags)

	home, err := projSvc.Create(ctx, user.ID, services.CreateProjectInput{Name: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	kitchen, err := projSvc.Create(ctx, user.ID, services.CreateProjectInput{Name: "Kitchen", ParentID: &home.ID, Sequential: true})
	if err != nil {
		t.Fatal(err)
	}
	if kitchen.Depth != 1 {
		t.Fatalf("expected depth 1, got %d", kitchen.Depth)
	}
	if _, err := projSvc.Create(ctx, user.ID, services.CreateProjectInput{Name: "Too deep", ParentID: &kitchen.ID}); err == nil {
		t.Fatal("third level should be rejected")
	}

	// Tags
	errands, err := tagSvc.Create(ctx, user.ID, "errands", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tagSvc.Create(ctx, user.ID, "Errands", ""); err == nil {
		t.Fatal("duplicate tag name should be rejected")
	}
	ids, err := tagSvc.EnsureByNames(ctx, user.ID, []string{"errands", "calls"})
	if err != nil || len(ids) != 2 || ids[0] != errands.ID {
		t.Fatalf("ensure by names: %v %v", ids, err)
	}

	// Tasks: inbox, project, sequential availability, defer
	inbox, err := taskSvc.Create(ctx, user.ID, services.CreateTaskInput{Title: "Capture me", Flagged: true, TagIDs: []string{errands.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if !inbox.IsAvailable || len(inbox.Tags) != 1 || inbox.Tags[0].Name != "errands" {
		t.Fatalf("inbox task should be available with one tag: %+v", inbox)
	}
	first, _ := taskSvc.Create(ctx, user.ID, services.CreateTaskInput{Title: "Buy paint", ProjectID: &kitchen.ID})
	second, _ := taskSvc.Create(ctx, user.ID, services.CreateTaskInput{Title: "Paint wall", ProjectID: &kitchen.ID})
	if !first.IsAvailable || second.IsAvailable {
		t.Fatalf("sequential project: only first task available (first=%v second=%v)", first.IsAvailable, second.IsAvailable)
	}
	future := time.Now().Add(48 * time.Hour)
	deferred, _ := taskSvc.Create(ctx, user.ID, services.CreateTaskInput{Title: "Later", ProjectID: &home.ID, DeferUntil: &future})
	if deferred.IsAvailable {
		t.Fatal("deferred task should not be available")
	}

	list := func(f domain.TaskFilter) []domain.Task {
		t.Helper()
		out, err := taskSvc.List(ctx, user.ID, f)
		if err != nil {
			t.Fatalf("list %+v: %v", f, err)
		}
		return out
	}
	if got := list(domain.TaskFilter{View: domain.ViewInbox}); len(got) != 1 || got[0].ID != inbox.ID {
		t.Fatalf("inbox view: %+v", got)
	}
	if got := list(domain.TaskFilter{View: domain.ViewAvailable}); len(got) != 2 {
		t.Fatalf("available view expected 2 (inbox + first sequential), got %d", len(got))
	}
	if got := list(domain.TaskFilter{View: domain.ViewFlagged}); len(got) != 1 {
		t.Fatalf("flagged view: %d", len(got))
	}
	if got := list(domain.TaskFilter{TagID: &errands.ID}); len(got) != 1 {
		t.Fatalf("tag filter: %d", len(got))
	}
	if got := list(domain.TaskFilter{ProjectID: &kitchen.ID}); len(got) != 2 {
		t.Fatalf("project filter: %d", len(got))
	}
	if got := list(domain.TaskFilter{Query: "wall"}); len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("search: %+v", got)
	}

	// Reorder: in a sequential project the first task in order is the available one
	reordered, err := taskSvc.Reorder(ctx, user.ID, []string{second.ID, first.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !reordered[0].IsAvailable || reordered[1].IsAvailable || reordered[0].ID != second.ID {
		t.Fatalf("after reorder the moved task should be available: %+v", reordered)
	}
	if _, err := taskSvc.Reorder(ctx, user.ID, []string{first.ID, "00000000-0000-0000-0000-000000000000"}); err == nil {
		t.Fatal("unknown id in reorder should be rejected")
	}
	if _, err = taskSvc.Reorder(ctx, user.ID, []string{first.ID, second.ID}); err != nil {
		t.Fatal(err)
	}

	// Project counts
	k, _ := projSvc.Get(ctx, user.ID, kitchen.ID)
	if k.RemainingTaskCount != 2 || k.AvailableTaskCount != 1 {
		t.Fatalf("kitchen counts remaining=%d available=%d", k.RemainingTaskCount, k.AvailableTaskCount)
	}
	h, _ := projSvc.Get(ctx, user.ID, home.ID)
	if h.RemainingTaskCount != 1 || h.AvailableTaskCount != 0 {
		t.Fatalf("home counts remaining=%d available=%d", h.RemainingTaskCount, h.AvailableTaskCount)
	}

	// Defer via natural language in the user's timezone
	u, _ := users.GetByID(ctx, user.ID)
	dt, parsed, err := taskSvc.Defer(ctx, u, inbox.ID, services.DeferInput{Input: "tomorrow"})
	if err != nil || dt.DeferUntil == nil || parsed == nil {
		t.Fatalf("defer: %v %+v", err, dt)
	}
	if dt.IsAvailable {
		t.Fatal("deferred to tomorrow should not be available")
	}
	if dt.DeferUntil.In(u.Location()).Hour() != 0 {
		t.Fatalf("defer should land at local midnight, got %s", dt.DeferUntil.In(u.Location()))
	}
	dt, _, err = taskSvc.Defer(ctx, u, inbox.ID, services.DeferInput{Clear: true})
	if err != nil || dt.DeferUntil != nil {
		t.Fatalf("clear defer: %v %+v", err, dt)
	}

	// Update: move to project, clear tags, set due + repeat
	due := time.Now().Add(24 * time.Hour)
	moved, err := taskSvc.Update(ctx, user.ID, inbox.ID, services.TaskPatch{
		SetProject: true, ProjectID: &home.ID, TagIDs: []string{},
		SetDue: true, DueAt: &due,
		SetRepeat: true, RepeatRule: &domain.RepeatRule{Every: 1, Unit: domain.RepeatWeek, From: domain.RepeatFromDue},
	})
	if err != nil {
		t.Fatal(err)
	}
	if moved.ProjectID == nil || *moved.ProjectID != home.ID || len(moved.Tags) != 0 || moved.RepeatRule == nil {
		t.Fatalf("update: %+v", moved)
	}
	if got := list(domain.TaskFilter{View: domain.ViewInbox}); len(got) != 0 {
		t.Fatalf("inbox should be empty after move, got %d", len(got))
	}

	// Complete: repeating task spawns next occurrence a week after the due date
	done, next, err := taskSvc.Complete(ctx, user.ID, moved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != domain.TaskCompleted || next == nil {
		t.Fatalf("complete: %+v next=%v", done, next)
	}
	if next.DueAt == nil || next.DueAt.Sub(due).Round(time.Minute) != 7*24*time.Hour {
		t.Fatalf("next due should be +1 week: %v", next.DueAt)
	}
	if got := list(domain.TaskFilter{View: domain.ViewCompleted}); len(got) != 1 {
		t.Fatalf("completed view: %d", len(got))
	}

	// Counts
	c, err := taskSvc.Counts(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Inbox != 0 || c.DueSoon != 0 || c.Overdue != 0 || c.ReviewDue != 0 {
		t.Fatalf("counts: %+v", c)
	}

	// Reviews: nothing due yet; force one due then mark reviewed
	dueList, upcoming, err := projSvc.Reviews(ctx, user.ID)
	if err != nil || len(dueList) != 0 || len(upcoming) != 2 {
		t.Fatalf("reviews: due=%d upcoming=%d err=%v", len(dueList), len(upcoming), err)
	}
	past := time.Now().Add(-time.Hour)
	h.NextReviewAt = &past
	if err := projects.Update(ctx, h); err != nil {
		t.Fatal(err)
	}
	dueList, _, _ = projSvc.Reviews(ctx, user.ID)
	if len(dueList) != 1 || dueList[0].ID != home.ID {
		t.Fatalf("home should be due for review: %+v", dueList)
	}
	c, _ = taskSvc.Counts(ctx, user.ID)
	if c.ReviewDue != 1 {
		t.Fatalf("review_due count: %d", c.ReviewDue)
	}
	reviewed, err := projSvc.MarkReviewed(ctx, user.ID, home.ID)
	if err != nil || reviewed.LastReviewedAt == nil || !reviewed.NextReviewAt.After(time.Now()) {
		t.Fatalf("mark reviewed: %v %+v", err, reviewed)
	}

	// Project status change cascades to availability; on_hold hides tasks
	onHold := domain.ProjectOnHold
	if _, err := projSvc.Update(ctx, user.ID, kitchen.ID, services.ProjectPatch{Status: &onHold}); err != nil {
		t.Fatal(err)
	}
	if got := list(domain.TaskFilter{View: domain.ViewAvailable}); len(got) != 1 {
		t.Fatalf("on-hold project tasks should be unavailable; available=%d", len(got))
	}

	// Delete project cascades tasks
	if err := projSvc.Delete(ctx, user.ID, home.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := taskSvc.Get(ctx, user.ID, next.ID); err != domain.ErrNotFound {
		t.Fatalf("task should be gone with its project: %v", err)
	}
	if _, err := projSvc.Get(ctx, user.ID, kitchen.ID); err != domain.ErrNotFound {
		t.Fatalf("child project should cascade: %v", err)
	}
}
