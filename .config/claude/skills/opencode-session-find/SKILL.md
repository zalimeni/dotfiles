---
name: opencode-session-find
description: Search OpenCode sessions across all projects by keyword, date, or project path. Use when user needs to find a session ID, especially when the original project directory is gone or they want a cross-project view.
license: MIT
compatibility: opencode
metadata:
  category: workflow
  tools: bash
---

## When to use me

- User wants to find an old OpenCode session but doesn't remember the project directory
- User wants to list sessions across all projects
- Original project directory was deleted or moved
- User wants to resume a session and needs the session ID

## Arguments

$ARGUMENTS

Optional search terms. Can be:
- A keyword to match against session titles (e.g., `auth refactor`)
- A project path fragment (e.g., `infragraph`)
- A date filter: `today`, `yesterday`, `7d`, `30d`, or `YYYY-MM-DD`
- Multiple terms separated by spaces — all are ANDed together
- No arguments = show 20 most recent sessions across all projects

## Instructions

### 1. Locate the database

The OpenCode SQLite database is at `~/.local/share/opencode/opencode.db`. Verify it exists before proceeding.

### 2. Build and run the query

Query the `session` and `project` tables. Always join on `project_id` to get the project worktree.

Base query:

```sql
SELECT
  s.id,
  s.title,
  s.directory,
  p.worktree,
  datetime(s.time_created / 1000, 'unixepoch', 'localtime') AS created,
  datetime(s.time_updated / 1000, 'unixepoch', 'localtime') AS updated
FROM session s
JOIN project p ON s.project_id = p.id
```

Apply filters from arguments:
- **Keyword**: `WHERE s.title LIKE '%keyword%'`
- **Project path fragment**: `WHERE (p.worktree LIKE '%fragment%' OR s.directory LIKE '%fragment%')`
- **Date filters**:
  - `today`: `WHERE date(s.time_updated / 1000, 'unixepoch', 'localtime') = date('now', 'localtime')`
  - `yesterday`: `WHERE date(s.time_updated / 1000, 'unixepoch', 'localtime') = date('now', '-1 day', 'localtime')`
  - `7d`: `WHERE s.time_updated > (strftime('%s', 'now', '-7 days') * 1000)`
  - `30d`: `WHERE s.time_updated > (strftime('%s', 'now', '-30 days') * 1000)`
  - `YYYY-MM-DD`: `WHERE date(s.time_updated / 1000, 'unixepoch', 'localtime') = 'YYYY-MM-DD'`

Always order by `s.time_updated DESC` and limit to 30 results unless user asks for more.

### 3. Present results

Display results as a table with columns: **Session ID**, **Title**, **Project**, **Last Updated**.

Truncate long titles at 60 chars. Show project as the last path component of `worktree` (or full path if ambiguous).

### 4. Offer to resume

After showing results, offer to resume a session. Determine which approach is safe:

**If the session's directory still exists on disk:**
- Offer to resume directly. Provide the exact command:
  ```
  opencode -s <session-id>
  ```
  Run from the session's directory (use `workdir` parameter).

**If the session's directory is gone:**
- Warn the user the original directory no longer exists
- Provide the command they can run manually from an appropriate directory:
  ```
  opencode -s <session-id>
  ```
- Suggest they `cd` to the project worktree first if it still exists, or pick a suitable directory

**Do NOT auto-resume** if:
- The directory is gone
- Multiple sessions matched and user hasn't picked one
- User only asked to search, not resume

### 5. Edge cases

- If no sessions match, say so and suggest broadening the search
- If the database doesn't exist, tell user OpenCode may not be installed or hasn't created any sessions yet
- Timestamps in the DB are Unix milliseconds — always divide by 1000 for SQLite datetime functions
