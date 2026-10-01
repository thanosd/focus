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
	// And back to the top.
	if _, err := projSvc.Update(ctx, user.ID, house.ID, services.ProjectPatch{SetParent: true, ParentID: nil}); err != nil {
		t.Fatal(err)
	}
}
