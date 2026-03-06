---
description: quick overview of bd project status
allowed-tools: Bash(bd *)
---

# BD Status Overview

## Overview

Quick project status snapshot — show what's ready, what's blocked, and overall progress. Use at session start, between work blocks, or anytime you need orientation.

## Arguments

$ARGUMENTS

Optional: epic ID to scope the status view. If not provided, shows full project status.

## Instructions

### 1. Gather Status

Run these commands and collect the output:

```bash
bd stats                    # Overall progress (open/closed/total)
bd ready                    # Issues ready to work on now
bd blocked                  # Issues waiting on dependencies
bd list --status in_progress  # Currently active work
```

If scoped to an epic:
```bash
bd show [epic-id]           # Epic details
bd dep tree [epic-id]       # Dependency structure
```

### 2. Present Summary

Output a concise status card:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  BD STATUS [scope or "all"]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Progress:    [closed]/[total] issues ([%])
  In Progress: [count] issues
  Ready:       [count] issues
  Blocked:     [count] issues

  Ready to work on:
    - [issue-id] (P[pri]) [title]
    - [issue-id] (P[pri]) [title]
    ...

  Blocked:
    - [issue-id] blocked by [blocker-ids]
    ...

  In Progress:
    - [issue-id] [title]
    ...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 3. Suggest Next Action

Based on the status, suggest what to do:
- If ready issues exist: "Run `/bdexecissue [highest-priority-ready-id]` to start work"
- If everything is blocked: "Resolve blockers first — see blocked list above"
- If everything is closed: "All work complete"
- If in-progress issues exist with no ready work: "Finish in-progress work before picking up new issues"
