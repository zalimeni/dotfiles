---
description: clean up a bdagent worktree and tmux session
argument-hint: <branch-name>
allowed-tools: Bash(git *), Bash(tmux *)
---

# BD Agent Cleanup

## Overview

Clean up a git worktree and tmux session previously created by `/bdagent`. Removes the worktree directory, deletes the local branch, and kills the tmux session.

## Arguments

$ARGUMENTS

**Required:**
- `<branch-name>`: The branch name used when creating the agent (same as the first argument to `/bdagent`)

**Example:**
```
/bdagent-cleanup feature-user-auth
```

## Instructions

### 1. Validate Argument

Verify the branch name is provided. If not, show usage and list existing worktrees to help the user pick one:

```bash
git worktree list
```

### 2. Kill tmux Session (if running)

The tmux session name matches the branch name (with `.` replaced by `_`):

```bash
session_name=$(echo "$branch_name" | tr . _)
tmux kill-session -t "$session_name" 2>/dev/null && echo "Killed tmux session: $session_name" || echo "No tmux session found: $session_name"
```

### 3. Remove Git Worktree

```bash
repo_root=$(git rev-parse --show-toplevel)
parent_dir=$(dirname "$repo_root")
worktree_dir="$parent_dir/$branch_name"

# Check the worktree exists
if git worktree list | grep -q "$worktree_dir"; then
    git worktree remove "$worktree_dir"
    echo "Removed worktree: $worktree_dir"
else
    echo "No worktree found at: $worktree_dir"
fi
```

If `git worktree remove` fails because of uncommitted changes, inform the user and suggest:
- `git worktree remove --force` to discard changes
- Or switching to the worktree to commit/stash first

### 4. Delete Local Branch

```bash
git branch -d "$branch_name" 2>/dev/null && echo "Deleted branch: $branch_name" || echo "Branch already deleted or has unmerged changes"
```

If `-d` fails (unmerged changes), warn the user and suggest `-D` only if they confirm the work was merged or is disposable.

### 5. Report

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  CLEANUP COMPLETE: $branch_name
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  tmux session:  [killed / not found]
  Worktree:      [removed / not found]
  Branch:        [deleted / not found / has unmerged changes]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Listing All Agent Worktrees

To see all active worktrees (useful if you forgot the branch name):

```bash
git worktree list
tmux list-sessions
```

## Error Handling

- **Worktree not found**: May have already been cleaned up. Skip and continue.
- **tmux session not found**: Agent may have already exited. Skip and continue.
- **Uncommitted changes in worktree**: Warn the user. Do NOT force-remove without confirmation.
- **Unmerged branch**: Warn and suggest checking if the work was pushed/merged before deleting.
