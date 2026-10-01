// Package importer loads an OmniFocus CSV export into Focus.
//
// OmniFocus exports one row per project or action with a dotted Task ID
// that encodes the hierarchy (1 = project, 1.2 = action, 1.2.3 = action
// inside an action group). Focus has no nested tasks, so action groups
// (actions with children) become second-level projects; everything else
// maps 1:1 onto tasks, tags and project status.
package importer

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/ports"
	"github.com/thanosd/focus/backend/internal/services"
)

// Row is one parsed CSV record.
type Row struct {
	ID          string
	Type        string // Project | Action
	Name        string
	Status      string // active | inactive (projects only)
	Project     string
	Context     string
	Start       *time.Time
	Due         *time.Time
	Completed   *time.Time
	Flagged     bool
	Notes       string
	Tags        []string
	parentID    string
	hasChildren bool
	collapsed   bool
}

// Plan is the resolved import: what will be created, in order.
type Plan struct {
	Projects []PlannedProject
	Tasks    []PlannedTask
	Tags     []string
	Warnings []string
}

// PlannedProject is a project (top-level or group) to create.
type PlannedProject struct {
	Key    string // CSV Task ID
	Parent string // CSV Task ID of the parent project ("" for top level)
	Name   string
	Note   string
	Status domain.ProjectStatus
}

// PlannedTask is a task to create.
type PlannedTask struct {
	Key        string
	ProjectKey string // CSV Task ID of the containing project/group
	Title      string
	Note       string
	Flagged    bool
	DeferUntil *time.Time
	DueAt      *time.Time
	Completed  *time.Time
	Tags       []string
}

const dateLayout = "2006-01-02 15:04:05 -0700"

// Parse reads the CSV and resolves the hierarchy.
func Parse(r io.Reader) (*Plan, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv has no data rows")
	}
	col := map[string]int{}
	for i, h := range records[0] {
		col[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))] = i
	}
	for _, required := range []string{"task id", "type", "name"} {
		if _, ok := col[required]; !ok {
			return nil, fmt.Errorf("csv is missing the %q column", required)
		}
	}
	get := func(rec []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	plan := &Plan{}
	var rows []*Row
	byID := map[string]*Row{}
	for n, rec := range records[1:] {
		if len(rec) == 0 || (len(rec) == 1 && rec[0] == "") {
			continue
		}
		row := &Row{
			ID:      get(rec, "task id"),
			Type:    get(rec, "type"),
			Name:    get(rec, "name"),
			Status:  strings.ToLower(get(rec, "status")),
			Project: get(rec, "project"),
			Context: get(rec, "context"),
			Flagged: get(rec, "flagged") == "1",
			Notes:   strings.TrimSpace(get(rec, "notes")),
		}
		if row.ID == "" || row.Name == "" {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("line %d: skipped row without id/name", n+2))
			continue
		}
		var perr error
		if row.Start, perr = parseDate(get(rec, "start date")); perr != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: bad start date: %v", row.ID, perr))
		}
		if row.Due, perr = parseDate(get(rec, "due date")); perr != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: bad due date: %v", row.ID, perr))
		}
		if row.Completed, perr = parseDate(get(rec, "completion date")); perr != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: bad completion date: %v", row.ID, perr))
		}
		if v := get(rec, "planned date"); v != "" {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: planned date %q is not supported and was dropped", row.ID, v))
		}
		if v := get(rec, "duration"); v != "" {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: duration %q is not supported and was dropped", row.ID, v))
		}
		row.Tags = splitTags(get(rec, "tags"))
		if row.Context != "" && !containsFold(row.Tags, row.Context) {
			row.Tags = append([]string{row.Context}, row.Tags...)
		}
		if i := strings.LastIndex(row.ID, "."); i >= 0 {
			row.parentID = row.ID[:i]
		}
		rows = append(rows, row)
		byID[row.ID] = row
	}
	children := map[string][]*Row{}
	for _, row := range rows {
		if p, ok := byID[row.parentID]; ok {
			p.hasChildren = true
			children[p.ID] = append(children[p.ID], row)
		}
	}
	// An action group with a single child of the same name is OmniFocus
	// noise (a task that was once turned into a group). Collapse it: the
	// child becomes a plain task in the group's parent, keeping its own
	// attributes and inheriting any tags the group carried.
	for _, row := range rows {
		if strings.EqualFold(row.Type, "Project") || !row.hasChildren {
			continue
		}
		kids := children[row.ID]
		if len(kids) != 1 || !strings.EqualFold(strings.TrimSpace(kids[0].Name), strings.TrimSpace(row.Name)) || kids[0].hasChildren {
			continue
		}
		child := kids[0]
		for _, t := range row.Tags {
			if !containsFold(child.Tags, t) {
				child.Tags = append(child.Tags, t)
			}
		}
		if child.Notes == "" {
			child.Notes = row.Notes
		}
		child.parentID = row.parentID
		row.collapsed = true
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: action group %q had one identically named child; imported as a single task", row.ID, row.Name))
	}

	tagSet := map[string]string{}
	for _, row := range rows {
		if row.collapsed {
			continue
		}
		switch {
		case strings.EqualFold(row.Type, "Project"):
			if row.parentID != "" {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: nested project %q imported as top-level (OmniFocus folders are not modelled)", row.ID, row.Name))
			}
			status := domain.ProjectActive
			switch row.Status {
			case "", "active":
			case "inactive", "on hold", "on_hold", "paused":
				status = domain.ProjectOnHold
			case "done", "completed":
				status = domain.ProjectCompleted
			case "dropped":
				status = domain.ProjectDropped
			default:
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: unknown project status %q, using active", row.ID, row.Status))
			}
			if row.Start != nil || row.Due != nil || len(row.Tags) > 0 || row.Flagged {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: project %q has dates/tags/flag that projects don't carry; dropped", row.ID, row.Name))
			}
			plan.Projects = append(plan.Projects, PlannedProject{Key: row.ID, Name: row.Name, Note: row.Notes, Status: status})

		case row.hasChildren:
			// Action group → second-level project. Its parent must be a
			// top-level project (OmniFocus allows deeper nesting; we don't).
			parent, ok := byID[row.parentID]
			if !ok || !strings.EqualFold(parent.Type, "Project") {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: action group %q is nested deeper than one level; flattening its tasks into the nearest project", row.ID, row.Name))
				continue
			}
			if len(row.Tags) > 0 || row.Start != nil || row.Due != nil || row.Flagged {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: action group %q became a sub-project; its own tags/dates/flag were dropped", row.ID, row.Name))
			}
			plan.Projects = append(plan.Projects, PlannedProject{Key: row.ID, Parent: row.parentID, Name: row.Name, Note: row.Notes, Status: domain.ProjectActive})

		default:
			projectKey := nearestProjectKey(row, byID)
			if projectKey == "" {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: task %q has no project row; importing into the inbox", row.ID, row.Name))
			}
			for _, t := range row.Tags {
				if _, seen := tagSet[strings.ToLower(t)]; !seen {
					tagSet[strings.ToLower(t)] = t
				}
			}
			plan.Tasks = append(plan.Tasks, PlannedTask{
				Key: row.ID, ProjectKey: projectKey, Title: row.Name, Note: row.Notes, Flagged: row.Flagged,
				DeferUntil: row.Start, DueAt: row.Due, Completed: row.Completed, Tags: row.Tags,
			})
		}
	}
	for _, t := range tagSet {
		plan.Tags = append(plan.Tags, t)
	}
	sort.Strings(plan.Tags)
	return plan, nil
}

// nearestProjectKey walks up the dotted ID until it finds a row that
// became a project (a Project row or a one-level action group).
func nearestProjectKey(row *Row, byID map[string]*Row) string {
	for id := row.parentID; id != ""; {
		p, ok := byID[id]
		if !ok {
			return ""
		}
		if strings.EqualFold(p.Type, "Project") {
			return p.ID
		}
		if p.hasChildren {
			gp, ok := byID[p.parentID]
			if ok && strings.EqualFold(gp.Type, "Project") {
				return p.ID
			}
		}
		id = p.parentID
	}
	return ""
}

func parseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{dateLayout, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("unrecognised date %q", s)
}

func splitTags(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" && !containsFold(out, p) {
			out = append(out, p)
		}
	}
	return out
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// Summary renders the plan for a dry run.
func (p *Plan) Summary() string {
	var b strings.Builder
	top, groups := 0, 0
	for _, pr := range p.Projects {
		if pr.Parent == "" {
			top++
		} else {
			groups++
		}
	}
	fmt.Fprintf(&b, "Projects: %d top-level, %d sub-projects (from action groups)\n", top, groups)
	fmt.Fprintf(&b, "Tasks: %d (%d flagged, %d deferred, %d with due dates, %d completed)\n",
		len(p.Tasks), count(p.Tasks, func(t PlannedTask) bool { return t.Flagged }),
		count(p.Tasks, func(t PlannedTask) bool { return t.DeferUntil != nil }),
		count(p.Tasks, func(t PlannedTask) bool { return t.DueAt != nil }),
		count(p.Tasks, func(t PlannedTask) bool { return t.Completed != nil }))
	fmt.Fprintf(&b, "Tags: %s\n", strings.Join(p.Tags, ", "))
	names := map[string]string{}
	for _, pr := range p.Projects {
		names[pr.Key] = pr.Name
	}
	for _, pr := range p.Projects {
		if pr.Parent == "" {
			fmt.Fprintf(&b, "\n%s [%s]\n", pr.Name, pr.Status)
		} else {
			fmt.Fprintf(&b, "\n%s / %s\n", names[pr.Parent], pr.Name)
		}
		for _, t := range p.Tasks {
			if t.ProjectKey == pr.Key {
				fmt.Fprintf(&b, "  - %s%s\n", t.Title, taskSuffix(t))
			}
		}
	}
	inbox := false
	for _, t := range p.Tasks {
		if t.ProjectKey == "" {
			if !inbox {
				fmt.Fprintf(&b, "\nInbox\n")
				inbox = true
			}
			fmt.Fprintf(&b, "  - %s%s\n", t.Title, taskSuffix(t))
		}
	}
	if len(p.Warnings) > 0 {
		fmt.Fprintf(&b, "\nWarnings:\n")
		for _, w := range p.Warnings {
			fmt.Fprintf(&b, "  ! %s\n", w)
		}
	}
	return b.String()
}

func taskSuffix(t PlannedTask) string {
	var parts []string
	if t.Flagged {
		parts = append(parts, "flagged")
	}
	if t.DeferUntil != nil {
		parts = append(parts, "defer "+t.DeferUntil.Format("2006-01-02"))
	}
	if t.DueAt != nil {
		parts = append(parts, "due "+t.DueAt.Format("2006-01-02"))
	}
	if len(t.Tags) > 0 {
		parts = append(parts, "#"+strings.Join(t.Tags, " #"))
	}
	if len(parts) == 0 {
		return ""
	}
	return "  (" + strings.Join(parts, ", ") + ")"
}

func count(ts []PlannedTask, f func(PlannedTask) bool) int {
	n := 0
	for _, t := range ts {
		if f(t) {
			n++
		}
	}
	return n
}

// Importer writes a Plan through the regular services so every rule
// (nesting limit, tag uniqueness, sort order) applies.
type Importer struct {
	Projects *services.ProjectService
	Tasks    *services.TaskService
	Tags     *services.TagService
	TaskRepo ports.TaskRepository
}

// Result counts what was created.
type Result struct {
	Projects, Tasks, Tags int
}

// Apply creates everything in the plan for the user.
func (im *Importer) Apply(ctx context.Context, userID string, plan *Plan) (*Result, error) {
	res := &Result{}
	existingTags, err := im.Tags.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	before := len(existingTags)
	if _, err := im.Tags.EnsureByNames(ctx, userID, plan.Tags); err != nil {
		return nil, fmt.Errorf("create tags: %w", err)
	}
	afterTags, err := im.Tags.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	res.Tags = len(afterTags) - before

	ids := map[string]string{}
	for _, pr := range plan.Projects {
		in := services.CreateProjectInput{Name: pr.Name, Note: pr.Note}
		if pr.Parent != "" {
			parentID, ok := ids[pr.Parent]
			if !ok {
				return nil, fmt.Errorf("project %q: parent %s was not created", pr.Name, pr.Parent)
			}
			in.ParentID = &parentID
		}
		p, err := im.Projects.Create(ctx, userID, in)
		if err != nil {
			return nil, fmt.Errorf("create project %q: %w", pr.Name, err)
		}
		if pr.Status != domain.ProjectActive {
			st := pr.Status
			if _, err := im.Projects.Update(ctx, userID, p.ID, services.ProjectPatch{Status: &st}); err != nil {
				return nil, fmt.Errorf("set status on %q: %w", pr.Name, err)
			}
		}
		ids[pr.Key] = p.ID
		res.Projects++
	}

	for _, t := range plan.Tasks {
		tagIDs, err := im.Tags.EnsureByNames(ctx, userID, t.Tags)
		if err != nil {
			return nil, err
		}
		in := services.CreateTaskInput{Title: t.Title, Note: t.Note, Flagged: t.Flagged, DeferUntil: t.DeferUntil, DueAt: t.DueAt, TagIDs: tagIDs}
		if t.ProjectKey != "" {
			pid, ok := ids[t.ProjectKey]
			if !ok {
				return nil, fmt.Errorf("task %q: project %s was not created", t.Title, t.ProjectKey)
			}
			in.ProjectID = &pid
		}
		created, err := im.Tasks.Create(ctx, userID, in)
		if err != nil {
			return nil, fmt.Errorf("create task %q: %w", t.Title, err)
		}
		if t.Completed != nil {
			created.Status = domain.TaskCompleted
			created.CompletedAt = t.Completed
			if err := im.TaskRepo.Update(ctx, created); err != nil {
				return nil, fmt.Errorf("mark %q completed: %w", t.Title, err)
			}
		}
		res.Tasks++
	}
	return res, nil
}
