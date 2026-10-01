package importer_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thanosd/focus/backend/internal/adapters"
	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/importer"
	"github.com/thanosd/focus/backend/internal/services"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
	"github.com/thanosd/focus/backend/internal/testdb"
)

func parseSample(t *testing.T) *importer.Plan {
	t.Helper()
	f, err := os.Open("testdata/sample.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	plan, err := importer.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestParse(t *testing.T) {
	plan := parseSample(t)
	if len(plan.Projects) != 5 { // 4 projects + 1 action group
		t.Fatalf("projects: %d %+v", len(plan.Projects), plan.Projects)
	}
	if len(plan.Tasks) != 8 {
		t.Fatalf("tasks: %d", len(plan.Tasks))
	}
	byKey := map[string]importer.PlannedTask{}
	for _, tk := range plan.Tasks {
		byKey[tk.Key] = tk
	}
	if byKey["2.1.1"].ProjectKey != "2.1" || !byKey["2.1.1"].Flagged || byKey["2.1.1"].DueAt == nil {
		t.Fatalf("group child: %+v", byKey["2.1.1"])
	}
	if !strings.Contains(byKey["2.1.1"].Note, `a "quote"`) || !strings.Contains(byKey["2.1.1"].Note, "\n") {
		t.Fatalf("multi-line quoted note not preserved: %q", byKey["2.1.1"].Note)
	}
	if byKey["2.2"].ProjectKey != "2" {
		t.Fatalf("plain task should be in the top project: %+v", byKey["2.2"])
	}
	attic := byKey["2.3.1"]
	if attic.ProjectKey != "2" || attic.DeferUntil == nil || attic.DeferUntil.Year() != 2026 || strings.Join(attic.Tags, ",") != "No Hurry,Home Improvement" {
		t.Fatalf("single-child group should collapse into one task with merged tags: %+v", attic)
	}
	if _, groupExists := byKey["2.3"]; groupExists {
		t.Fatal("collapsed group must not become a task")
	}
	if got := byKey["1.2"].Tags; len(got) != 2 || got[0] != "Phone" || got[1] != "Home" {
		t.Fatalf("tags with context: %v", got)
	}
	if byKey["1.3"].Completed == nil {
		t.Fatal("completion date lost")
	}
	if strings.Join(plan.Tags, ",") != "Computer,Home,Home Improvement,No Hurry,Phone" {
		t.Fatalf("tag set: %v", plan.Tags)
	}
	var wind importer.PlannedProject
	for _, p := range plan.Projects {
		if p.Name == "Wind Meter" {
			wind = p
		}
	}
	if wind.Status != domain.ProjectOnHold {
		t.Fatalf("inactive should map to on_hold: %+v", wind)
	}
	if !strings.Contains(plan.Summary(), "Maintain House / Recurring") {
		t.Fatalf("summary: %s", plan.Summary())
	}
}

func TestApply(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	users := adapters.NewUserRepository(db)
	projects := adapters.NewProjectRepository(db)
	tasks := adapters.NewTaskRepository(db)
	tags := adapters.NewTagRepository(db)
	user, err := users.UpsertFromGoogle(ctx, "import@example.com", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	im := &importer.Importer{
		Projects: services.NewProjectService(projects, tasks),
		Tasks:    services.NewTaskService(tasks, projects, dateparse.NewParser(nil)),
		Tags:     services.NewTagService(tags),
		TaskRepo: tasks,
	}
	res, err := im.Apply(ctx, user.ID, parseSample(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Projects != 5 || res.Tasks != 8 || res.Tags != 5 {
		t.Fatalf("result: %+v", res)
	}
	all, _ := projects.List(ctx, user.ID, []domain.ProjectStatus{domain.ProjectActive, domain.ProjectOnHold})
	var recurring *domain.Project
	for i := range all {
		if all[i].Name == "Recurring" {
			recurring = &all[i]
		}
	}
	if recurring == nil || recurring.Depth != 1 || recurring.RemainingTaskCount != 2 {
		t.Fatalf("Recurring sub-project: %+v", recurring)
	}
	completed, _ := tasks.List(ctx, user.ID, domain.TaskFilter{View: domain.ViewCompleted})
	if len(completed) != 1 || completed[0].Title != "Already done" {
		t.Fatalf("completed: %+v", completed)
	}
	flagged, _ := tasks.List(ctx, user.ID, domain.TaskFilter{View: domain.ViewFlagged})
	if len(flagged) != 2 {
		t.Fatalf("flagged: %d", len(flagged))
	}
	avail, _ := tasks.List(ctx, user.ID, domain.TaskFilter{View: domain.ViewAvailable})
	// Book flights (deferred, past), Call plumber, Change filters, Paint stairs;
	// Water bills is deferred to 2026-10-09; Calibrate is in an on-hold project.
	for _, tk := range avail {
		if tk.Title == "Calibrate" {
			t.Fatal("task in on-hold project should not be available")
		}
	}
}

// TestApplyRealExport imports a real OmniFocus export when FOCUS_IMPORT_CSV
// points at one — a rehearsal against a throwaway Postgres before running
// the import for real. Skipped otherwise.
func TestApplyRealExport(t *testing.T) {
	path := os.Getenv("FOCUS_IMPORT_CSV")
	if path == "" {
		t.Skip("FOCUS_IMPORT_CSV not set")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	plan, err := importer.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.Open(t)
	ctx := context.Background()
	users := adapters.NewUserRepository(db)
	projects := adapters.NewProjectRepository(db)
	tasks := adapters.NewTaskRepository(db)
	tags := adapters.NewTagRepository(db)
	user, err := users.UpsertFromGoogle(ctx, "rehearsal@example.com", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	im := &importer.Importer{
		Projects: services.NewProjectService(projects, tasks),
		Tasks:    services.NewTaskService(tasks, projects, dateparse.NewParser(nil)),
		Tags:     services.NewTagService(tags),
		TaskRepo: tasks,
	}
	res, err := im.Apply(ctx, user.ID, plan)
	if err != nil {
		t.Fatal(err)
	}
	if res.Projects != len(plan.Projects) || res.Tasks != len(plan.Tasks) {
		t.Fatalf("plan/result mismatch: %+v vs %d projects %d tasks", res, len(plan.Projects), len(plan.Tasks))
	}
	all, _ := tasks.List(ctx, user.ID, domain.TaskFilter{View: domain.ViewAll})
	avail, _ := tasks.List(ctx, user.ID, domain.TaskFilter{View: domain.ViewAvailable})
	inbox, _ := tasks.List(ctx, user.ID, domain.TaskFilter{View: domain.ViewInbox})
	counts, _ := tasks.Counts(ctx, user.ID, time.Now())
	t.Logf("imported %d projects, %d tasks, %d tags; active=%d available=%d inbox=%d flagged=%d overdue=%d",
		res.Projects, res.Tasks, res.Tags, len(all), len(avail), len(inbox), counts.Flagged, counts.Overdue)
}
