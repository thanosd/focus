# Importing from OmniFocus

Export from OmniFocus with **File → Export… → CSV**, then run the importer
built into the backend binary. It understands the standard export columns
(Task ID, Type, Name, Status, Project, Context, Start Date, Planned Date,
Due Date, Completion Date, Duration, Flagged, Notes, Tags).

## Mapping

| OmniFocus                              | Focus                                           |
| -------------------------------------- | ----------------------------------------------- |
| Project row                            | top-level project                               |
| Project status `active` / `inactive`   | `active` / `on_hold`                            |
| Action group (an action with children) | second-level project under its project          |
| Action                                 | task in the nearest project                     |
| Start Date                             | `defer_until`                                   |
| Due Date                               | `due_at`                                        |
| Completion Date                        | task marked `completed` with that timestamp     |
| Flagged `1`                            | `flagged`                                       |
| Notes (multi-line)                     | `note`                                          |
| Context + Tags                         | tags (created on the fly; context is merged in) |
| Planned Date, Duration                 | not modelled; a warning is printed if present   |
| Repeat rules                           | not in the export; set them in Focus afterwards |

An action group with a single identically named child collapses into one
task. Anything dropped (tags on a group, dates on a project) is listed in
the warnings so nothing disappears silently.

## Run it

Always rehearse first — `--dry-run` parses the file and prints the full
plan without connecting to a database:

```bash
go run ./backend/cmd/focus import-omnifocus ~/omnifocus-export.csv --dry-run
```

Then import for real. `DATABASE_URL` selects the target; the account is
looked up (or created) by email:

```bash
# Against production over Tailscale (DATABASE_URL from the cosmic/focus/production secret)
DATABASE_URL='postgresql://focus_production:...@<rds>:5432/focus_production?sslmode=require' \
  go run ./backend/cmd/focus import-omnifocus ~/omnifocus-export.csv \
  --user thanos.diacakis@cosmicteacups.com
```

The importer refuses to run into an account that already has projects or
tasks; pass `--allow-existing` to override (it does not de-duplicate).

Alternatively, from inside the cluster with the already-deployed image:

```bash
export KUBECONFIG=$PWD/kubeconfig.yaml
POD=$(kubectl get pod -n focus -l app.kubernetes.io/component=backend -o jsonpath='{.items[0].metadata.name}')
kubectl cp ~/omnifocus-export.csv focus/$POD:/tmp/export.csv
kubectl exec -n focus $POD -- /app/focus import-omnifocus /tmp/export.csv --user thanos.diacakis@cosmicteacups.com
```

## Rehearsing against a throwaway database

`FOCUS_IMPORT_CSV=~/omnifocus-export.csv go test -run TestApplyRealExport -v ./backend/internal/importer/`
applies the real file to an embedded Postgres and prints the resulting counts.

## After importing

- Set your timezone under **Settings** (OmniFocus exports UTC timestamps;
  Pacific midnight shows as 07:00/08:00 UTC and displays correctly once the
  timezone is set).
- Recreate repeat rules on the tasks that had them in OmniFocus
  (e.g. monthly invoices, water bills); the export only contains the
  current instance of each.
