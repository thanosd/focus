package adapters_test

import (
	"context"
	"testing"

	"github.com/thanosd/focus/backend/internal/adapters"
	"github.com/thanosd/focus/backend/internal/services"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
	"github.com/thanosd/focus/backend/internal/testdb"
)

// A top-level project that already has sub-projects must be movable under
// a new top-level bucket: bucket > project > sub-project is three levels.
func TestMoveProjectWithChildrenUnderBucket(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	users := adapters.NewUserRepository(db)
	projects := adapters.NewProjectRepository(db)
	tasks := adapters.NewTaskRepository(db)
	user, _ := users.UpsertFromGoogle(ctx, "move@example.com", "", "", "")
	projSvc := services.NewProjectService(projects, tasks)
	taskSvc := services.NewTaskService(tasks, projects, dateparse.NewParser(nil))

	house, _ := projSvc.Create(ctx, user.ID, services.CreateProjectInput{Name: "Maintain House"})
	recurring, _ := projSvc.Create(ctx, user.ID, services.CreateProjectInput{Name: "Recurring", ParentID: &house.ID})
	_, _ = taskSvc.Create(ctx, user.ID, services.CreateTaskInput{Title: "Water bills", ProjectID: &recurring.ID})
	personal, _ := projSvc.Create(ctx, user.ID, services.CreateProjectInput{Name: "Personal"})

	moved, err := projSvc.Update(ctx, user.ID, house.ID, services.ProjectPatch{SetParent: true, ParentID: &personal.ID})
	if err != nil {
		t.Fatalf("moving a project with sub-projects under a bucket must work: %v", err)
	}
	if moved.Depth != 1 {
		t.Fatalf("depth after move: %d", moved.Depth)
	}
	rec, _ := projSvc.Get(ctx, user.ID, recurring.ID)
	if rec.Depth != 2 {
		t.Fatalf("child depth after move: %d", rec.Depth)
	}
	// Sibling reorder: swap two top-level projects.
	ordered, err := projSvc.Reorder(ctx, user.ID, []string{personal.ID, house.ID})
	if err != nil {
		t.Fatal(err)
	}
	if ordered[0].ID != personal.ID || ordered[0].SortOrder != 0 || ordered[1].SortOrder != 1 {
		t.Fatalf("reorder: %+v", ordered)
	}
	list, _ := projSvc.List(ctx, user.ID, "")
	if list[0].ID != personal.ID {
		t.Fatalf("list should honour sort_order: %s first", list[0].Name)
	}
	// And back to the top.
	if _, err := projSvc.Update(ctx, user.ID, house.ID, services.ProjectPatch{SetParent: true, ParentID: nil}); err != nil {
		t.Fatal(err)
	}
}

// "First of every month" style repeats stay on the 1st however late the
// task is completed, and setting one on an undated task pins a due date.
func TestAnchoredRepeat(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	users := adapters.NewUserRepository(db)
	projects := adapters.NewProjectRepository(db)
	tasks := adapters.NewTaskRepository(db)
	user, _ := users.UpsertFromGoogle(ctx, "repeat@example.com", "", "", "")
	_ = users.UpdateTimezone(ctx, user.ID, "America/Los_Angeles")
	user, _ = users.GetByID(ctx, user.ID)
	taskSvc := services.NewTaskService(tasks, projects, dateparse.NewParser(nil))

	bills, _ := taskSvc.Create(ctx, user.ID, services.CreateTaskInput{Title: "Pay bills"})
	set, parsed, err := taskSvc.SetRepeat(ctx, user, bills.ID, services.RepeatInput{Input: "first of every month"})
	if err != nil {
		t.Fatal(err)
	}
	if parsed == nil || parsed.Description != "monthly on the 1st" || set.RepeatRule == nil || set.RepeatRule.DayOfMonth == nil {
		t.Fatalf("set repeat: %+v %+v", parsed, set.RepeatRule)
	}
	if set.DueAt == nil || set.DueAt.In(user.Location()).Day() != 1 || set.DueAt.In(user.Location()).Hour() != 17 {
		t.Fatalf("anchored repeat should pin a due date on the 1st at 17:00 local: %v", set.DueAt)
	}
	firstDue := *set.DueAt

	// Complete it late: the next one is still on the 1st of a later month.
	done, next, err := taskSvc.Complete(ctx, user.ID, bills.ID)
	if err != nil || done.Status != "completed" || next == nil || next.DueAt == nil {
		t.Fatalf("complete: %v %+v %+v", err, done, next)
	}
	nd := next.DueAt.In(user.Location())
	if nd.Day() != 1 || !next.DueAt.After(firstDue) {
		t.Fatalf("next due should be the 1st of a following month, got %v", nd)
	}

	// Clear it.
	cleared, _, err := taskSvc.SetRepeat(ctx, user, next.ID, services.RepeatInput{Clear: true})
	if err != nil || cleared.RepeatRule != nil {
		t.Fatalf("clear: %v %+v", err, cleared.RepeatRule)
	}
	if _, _, err := taskSvc.SetRepeat(ctx, user, next.ID, services.RepeatInput{Input: "when pigs fly"}); err == nil {
		t.Fatal("nonsense should be rejected")
	}
}
