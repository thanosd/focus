# Data Model

Schema of record: `backend/migrations/`. This is the orientation.

## Tables

| Table          | Purpose                                                                            |
| -------------- | ---------------------------------------------------------------------------------- |
| `users`        | Google accounts (`email` unique, `google_sub`, `timezone`)                         |
| `sessions`     | Browser sessions (30 days)                                                         |
| `oauth_states` | Sign-in nonces (10 minutes)                                                        |
| `api_tokens`   | MCP bearer tokens — only `token_hash` (SHA-256) is stored                          |
| `projects`     | `parent_id` for nesting (3 levels), `status`, `sequential`, review fields          |
| `tasks`        | `project_id` NULL = inbox; `flagged`, `defer_until`, `due_at`, `repeat_rule` JSONB |
| `tags`         | Unique per `(user_id, name)`                                                       |
| `task_tags`    | Many-to-many                                                                       |

All timestamps are `TIMESTAMPTZ`. Deleting a project cascades to its
child projects and all of their tasks. Deleting a user cascades everything.

## Statuses

- Task: `active` → `completed` | `dropped` (and back via reopen)
- Project: `active`, `on_hold`, `completed`, `dropped`

## Availability

A task is **available** when all of these hold:

1. `status = 'active'`
2. `defer_until IS NULL OR defer_until <= now()`
3. it is in the inbox, **or** its project is `active`
4. if the project is `sequential`, it is the first active task by
   `(sort_order, created_at)`

The rule lives in one SQL expression (`taskBase` in
`adapters/postgres_tasks.go`) and the project count subquery mirrors it, so
list views, badges and project counts can't disagree.

## Views (`GET /api/tasks?view=`)

| view        | filter                          | order               |
| ----------- | ------------------------------- | ------------------- |
| `inbox`     | active, no project              | sort order          |
| `available` | the rule above (default)        | sort order          |
| `flagged`   | active, flagged                 | due date, then sort |
| `due`       | active, has due date            | due date            |
| `completed` | completed or dropped (last 200) | most recent first   |
| `all`       | every active task               | sort order          |

`project_id`, `tag_id` and `q` (title/note search) combine with any view.

## Repeats

`repeat_rule = {"every": N, "unit": day|week|month|year, "from": completion|due,
"weekdays"?: [0-6], "day_of_month"?: 1-31 | -1}`.

Two optional anchors pin occurrences to the calendar: `weekdays` (weekly
rules, e.g. Mon/Wed/Fri) and `day_of_month` (monthly rules; `-1` = last
day, and days past a month's end clamp to its last day). `RepeatRule.Next`
computes the next occurrence honouring both; `Upcoming` previews several.

Rules can be set in plain English (`POST /api/tasks/{id}/repeat {input}` or
`POST /api/repeats/parse`): "first of every month", "every other friday",
"weekdays", "every 2 weeks after completion", "yearly on jan 1". The
rule-based parser (`services/repeatparse`) covers that vocabulary; Claude is
the fallback for anything else. Calendar-anchored phrases default to
`from: due`; bare intervals default to `from: completion`. Setting an
anchored rule on an undated task pins `due_at` to the first occurrence.

Every task response carries `repeat_description` and `next_occurrences`
(the next three dates if it were completed now) so the UI can show what's
coming back before you tick it off.

On completion the task is marked `completed` and a new active task is
created with the same title, note, project, flag and tags:

- `from: completion` — the next date is `Next(completion time)`; a defer
  date keeps its time of day, and a due date keeps its gap from the defer.
- `from: due` — `due_at` advances from its previous value (catching up past
  the completion time if it's overdue); the defer date keeps its gap. A
  monthly "1st" task stays on the 1st however late you finish it.

## Reviews

Each project has `review_interval_days` (default 7) and `next_review_at`
(initially creation + interval). `GET /api/reviews` returns active/on-hold
projects with `next_review_at <= now` as **due** (oldest first) and the rest
as **upcoming**. Pure containers (projects that have sub-projects and no
tasks of their own) are skipped: they're navigation, not commitments. `POST /api/projects/{id}/review` stamps `last_reviewed_at`
and sets `next_review_at = now + interval`.

## Dates and timezones

The user's IANA `timezone` (Settings) drives natural-language parsing:
defer phrases resolve to **00:00 local**, due phrases to **17:00 local**
unless the phrase carries a time. Clients can override per request with
`timezone` on `/api/dates/parse` and `/api/tasks/{id}/defer`.
