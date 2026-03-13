---
name: bdloop
description: execute-review-fix loop until review passes
argument-hint: [scope]
allowed-tools: Skill, Task, Bash(bd:*), Bash(git *)
---

# BD Execute-Review-Fix Loop

## Overview

Self-correcting feedback loop: execute ready bd issues, review the resulting code, create fix issues from review findings, and re-execute until the review passes clean. Combines `/bdexecplan`, the `review` agent, and `/bdplan` into an automated quality loop.

## Arguments

$ARGUMENTS

Optional: epic ID or search term to scope which issues to work on. Passed through to `/bdexecplan` and used to filter `bd ready`.

## Loop Architecture

```
bdloop [scope]
  ├── Pre-loop: check clean tree, capture git baseline, verify ready work
  │
  ├── Iteration N:
  │   ├── Record iteration baseline (git rev-parse HEAD)
  │   ├── /bdexecplan [scope]
  │   ├── Check for changes since baseline (git log)
  │   ├── /bdreview [ITER_BASELINE] Iteration [N]
  │   │   ├── Review agent (scoped to diff)
  │   │   ├── Categorize findings
  │   │   ├── PASS → done, or NEEDS FIXES → /bdplan
  │   │   └── Check ready work
  │   └── Oscillation check (cross-iteration comparison)
  │
  └── Final summary report
```

## Instructions

### 1. Pre-Loop Setup

#### Check for Clean Working Tree

Before starting, verify no uncommitted changes exist:

```bash
git status --porcelain
```

If the working tree is dirty:
1. Warn the user: "Working tree has uncommitted changes."
2. Ask whether to stash them (`git stash push -m "bdloop: stash before execution"`) or abort.
3. Do NOT proceed with a dirty tree — bdexecissue makes commits, and uncommitted changes would be mixed in.

#### Sync with Remote

Fetch the latest remote state so that baselines and diffs are accurate. This prevents reviewing commits that are already merged upstream or missing recently-pushed work.

```bash
git fetch origin
```

If the current branch is behind its upstream, rebase to incorporate remote changes before starting work:

```bash
git pull --rebase
```

If the pull fails (e.g., conflicts), warn the user and abort — do not start the loop on a diverged branch.

#### Capture VCS Baseline

Record the current git commit SHA before any work begins:

```bash
git rev-parse HEAD
```

Store this as `LOOP_BASELINE` — used to scope the final summary.

#### Verify Ready Work Exists

```bash
bd ready
```

If scoped, filter to relevant issues. If nothing is ready, exit immediately with a message — there is nothing to do.

Initialize iteration counter to 0 and max iterations to **5**.

### 2. Iteration Loop

Repeat until a stopping condition is met:

#### Step A: Increment and Record Iteration Baseline

Increment the iteration counter. Record the current git commit SHA:

```bash
git rev-parse HEAD
```

Store as `ITER_BASELINE` for this iteration.

Output an iteration header:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⟳ ITERATION [N] of 5
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### Step B: Execute Plan

Invoke `/bdexecplan` with the scope argument:

```
Skill("bdexecplan", args="[scope]")
```

This runs all ready issues in the scope through `/bdexecissue`.

#### Step C: Check for Changes

After execution completes, check whether any new commits were produced:

```bash
git log $ITER_BASELINE..HEAD --oneline
```

If no new changes exist since the iteration baseline, exit the loop — execution produced nothing to review.

#### Step D: Review and Plan Fixes

Invoke `/bdreview` with the iteration baseline and label:

```
Skill("bdreview", args="[ITER_BASELINE] Iteration [N]")
```

This handles: review agent invocation, finding categorization, result card output, `/bdplan` delegation for actionable findings, and ready-work check.

Parse the review result card output from `/bdreview`:
- If verdict is **PASS** (zero Critical + zero Recommendations), exit the loop.
- If verdict is **NEEDS FIXES** but no new ready issues were created, exit the loop — there is nothing more to execute.

#### Step E: Oscillation Check

If the findings in this iteration are substantially similar to the previous iteration's findings, exit the loop with a warning — fixes are not converging. Compare finding descriptions; if >50% overlap, treat as oscillating.

#### Step F: Verify Ready Work for Next Iteration

If `/bdreview` reported no new ready issues, exit the loop — nothing can progress.

#### Step G: Continue

Loop back to Step A for the next iteration.

### 3. Max Iteration Guard

If the iteration counter reaches 5, exit the loop regardless of review status. Output a warning:

```
⚠ MAX ITERATIONS (5) REACHED — exiting loop.
  Review still has findings. Manual attention needed.
```

### 4. Final Summary Report

After exiting the loop (for any reason), output a summary:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  BDLOOP COMPLETE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Iterations:      [N]
  Issues executed:  [total count across all iterations]
  Exit reason:      [clean review / no changes / no new issues /
                     max iterations / oscillating fixes / no ready work]

  Iteration breakdown:
    1: Executed [X] issues, review found [Y] critical, [Z] recommendations
    2: Executed [X] issues, review found [Y] critical, [Z] recommendations
    ...

  Remaining suggestions (informational):
    - [any Suggestion-level findings from final review]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Stopping Conditions

Any of these triggers an exit:

| Condition                                      | Exit Reason                        |
| ---------------------------------------------- | ---------------------------------- |
| Zero Critical + zero Recommendations in review | `clean review`                     |
| No new commits after `/bdexecplan`             | `no changes`                       |
| No new ready issues after `/bdplan`            | `no new issues`                    |
| Iteration counter reaches 5                    | `max iterations`                   |
| Findings repeat across consecutive iterations  | `oscillating fixes`                |
| No ready work at start of loop                 | `no ready work`                    |
| Review agent fails/errors                      | `review error` (exit with warning) |

## Edge Cases

- **Review agent failure**: If the review Task errors or returns unusable output, exit the loop with a `review error` reason. Report what's known and recommend manual review.
- **Empty scope**: If `bd ready` returns nothing (or nothing matching scope) at any point, exit cleanly.
- **Oscillating fixes**: If two consecutive iterations produce >50% similar findings, exit with `oscillating fixes`. The fixes are not converging and human judgment is needed.
- **bdplan creates blocked issues**: If all new issues are blocked by existing open work, exit with `no new issues` since nothing can progress.

## Best Practices

1. **Stay lightweight** — orchestrator only tracks state, delegates all real work
2. **Trust the components** — `/bdexecplan` handles execution, review agent handles review, `/bdplan` handles planning
3. **Scope reviews narrowly** — only review each iteration's diff, not the entire codebase
4. **Report clearly** — keep the user informed with iteration cards and the final summary
5. **Exit early** — prefer stopping cleanly over grinding through marginal fixes
