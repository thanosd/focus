# MCP server

The backend serves an MCP endpoint at `/mcp` (streamable HTTP, stateless,
JSON responses) so Claude can read and edit your tasks through the same
services as the web UI.

## Connect

1. In the app, **Settings → API tokens → Create token**. Copy the `fcs_…`
   secret; it is shown once.
2. Claude Code:

   ```bash
   claude mcp add --transport http focus https://focus-api.cosmicteacups.com/mcp \
     --header "Authorization: Bearer fcs_..."
   ```

   Any other MCP client works the same way: URL + bearer header.

## Tools

| Tool                    | What it does                                                        |
| ----------------------- | ------------------------------------------------------------------- |
| `list_tasks`            | Views: `inbox`, `available` (default), `flagged`, `due`, `completed`, `all`; filter by project (name or id), tag, search |
| `get_task`              | Full task                                                           |
| `create_task`           | Title, note, project (name or id), flagged, `defer`, `due`, `repeat`, tag names (auto-created) |
| `update_task`           | Partial update; empty string clears a date/repeat; `project: "inbox"` moves back |
| `complete_task`         | Completes; returns `next_task` for repeats                          |
| `drop_task` / `reopen_task` / `delete_task` |                                                 |
| `defer_task`            | Natural language: "1w", "next monday", "in 3 days", "mid october"   |
| `list_projects`         | Flat list with `parent_id`, counts, review dates                    |
| `get_project`           | Project + active tasks + children                                   |
| `create_project`        | Optional parent (name or id), sequential, review interval           |
| `update_project`        | Name, note, status, sequential, review interval                     |
| `list_reviews`          | Due and upcoming reviews                                            |
| `mark_project_reviewed` |                                                                     |
| `list_tags` / `create_tag` |                                                                  |
| `parse_date`            | Resolve a phrase without changing anything                          |
| `get_counts`            | Badge counts                                                        |

Dates in `create_task` / `update_task` / `defer_task` accept natural
language or RFC 3339. Repeat rules accept "every 2 weeks", "monthly from
due", "daily". All timestamps are returned in the user's timezone.

## Auth model

The bearer token resolves to a user (`api_tokens.token_hash`); every tool
reads `req.Extra.TokenInfo.UserID` and calls the services with that user
ID, so data isolation is identical to the web app. Errors from services are
returned as tool errors (`isError: true`) with the validation message so the
model can self-correct ("couldn't understand \"when pigs fly\"").

## Local

```bash
claude mcp add --transport http focus-local http://localhost:8000/mcp \
  --header "Authorization: Bearer fcs_..."
```
