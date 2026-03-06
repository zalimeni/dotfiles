---
description: review code changes since a baseline commit and plan fixes
argument-hint: <target> [label]
allowed-tools: Skill, Task, Bash(bd:*), Bash(git *), Bash(gh *)
---

# BD Review

## Overview

Review code changes for a given target (commit, range, or PR), categorize findings by severity, and create fix issues via `/bdplan` for actionable findings. Can be used standalone after manual coding, to review a PR, or invoked as part of `/bdloop`.

## Arguments

$ARGUMENTS

- `target` (required): What to review. One of:
  - **Single SHA** — diffs `SHA..HEAD` (e.g., `abc1234`)
  - **SHA range** — diffs the range as-is (e.g., `abc1234..def5678`)
  - **PR reference** — diffs the pull request (e.g., `#123`, `123`, or a GitHub PR URL)
- `label` (optional): Context label for output cards (e.g., "Iteration 2")

## Instructions

### 1. Resolve Target

Detect the input type and resolve it into `DIFF_CMD` and `LOG_CMD` variables used by later steps.

| Input | Detection | DIFF_CMD | LOG_CMD |
| ----- | --------- | -------- | ------- |
| PR ref (`#N`, bare number, or `github.com/.../pull/N` URL) | matches `#?\d+$` or GH PR URL | `gh pr diff N` | `gh pr view N` |
| SHA range (`A..B`) | contains `..` | `git diff A..B` | `git log A..B --oneline` |
| Single SHA | default | `git diff SHA..HEAD` | `git log SHA..HEAD --oneline` |

**Validate the resolved target:**

- **PR**: `gh pr view N --json number` must succeed. If the PR is not found, exit with an error.
- **SHA range**: `git cat-file -t A` and `git cat-file -t B` must both resolve to commits.
- **Single SHA**: `git cat-file -t SHA` must resolve to a commit.

If validation fails, exit with an error message.

### 2. Check for Changes

Run `LOG_CMD` to verify there are changes to review.

- **PR**: `gh pr view N --json additions,deletions` — if both are 0, no changes.
- **SHA range / single SHA**: if the log output is empty, no changes.

If no changes, output a message and exit — nothing to review.

```
┌─ REVIEW RESULT [label] ────────────────
│ No changes found for [target].
│ Verdict: SKIP
└─────────────────────────────────────────
```

### 3. Run Review Agent

Invoke the review agent scoped to the resolved target:

**For SHA-based targets (single SHA or range):**

```
Task(
  description="Review changes for [target]",
  subagent_type="review",
  prompt="Review the code changes for [target].

Use this command to see the diff:
  [DIFF_CMD]

Use this command to see the commit log:
  [LOG_CMD]

Review all changed files thoroughly for correctness, security, best practices, error handling, and architecture."
)
```

**For PR targets:**

```
Task(
  description="Review PR #[N]",
  subagent_type="review",
  prompt="Review pull request #[N].

Use this command to see the PR details and description:
  gh pr view [N]

Use this command to see the diff:
  gh pr diff [N]

Use this command to see the changed files:
  gh pr diff [N] --name-only

IMPORTANT: This is a LOCAL-ONLY review. Do NOT post comments, reviews, or annotations to the pull request on GitHub. Do NOT use gh pr review, gh pr comment, or any command that writes to the remote PR. Only read the PR data and return your findings as text output.

Review all changed files thoroughly for correctness, security, best practices, error handling, and architecture."
)
```

**Local-only rule**: This skill MUST NOT post comments, reviews, or annotations to GitHub. All output is local. Never use `gh pr review`, `gh pr comment`, or any other `gh` subcommand that writes to a PR. Only read commands (`gh pr view`, `gh pr diff`) are permitted.

### 4. Categorize Findings

Parse the review agent's response. Count findings by category:

- **Critical** — must-fix issues (bugs, security, data integrity)
- **Recommendations** — should-fix improvements (performance, architecture, best practices)
- **Suggestions** — nice-to-have (informational only, do NOT trigger fixes)

### 5. Output Review Result Card

```
┌─ REVIEW RESULT [label] ────────────────
│ Critical:        [count]
│ Recommendations: [count]
│ Suggestions:     [count]
│ Verdict:         [PASS / NEEDS FIXES]
└─────────────────────────────────────────
```

If zero Critical AND zero Recommendations → verdict is PASS. Done.

### 6. Create Fix Issues (if NEEDS FIXES)

If there are Critical or Recommendation findings, invoke `/bdplan` to create fix issues. Pass only Critical and Recommendation findings — not Suggestions:

```
Skill("bdplan", args="Fix issues from review [label]:

[paste Critical and Recommendation findings here, not Suggestions]")
```

### 7. Check for Ready Work

```bash
bd ready
```

Report whether new ready issues were created.

### 8. Output Summary

```
┌─ REVIEW SUMMARY [label] ───────────────
│ Verdict:     [PASS / NEEDS FIXES]
│ Critical:    [count]
│ Recommendations: [count]
│ Suggestions: [count]
│ New ready issues: [yes/no]
└─────────────────────────────────────────
```

## Edge Cases

- **Review agent failure**: If the review Task errors or returns unusable output, exit with a warning and recommend manual review.
- **No changes**: Exit early with SKIP verdict (Step 2).
- **Invalid target**: Exit with error (Step 1) — SHA doesn't exist, PR not found, etc.
- **Closed/merged PR**: Still reviewable — `gh pr diff` works on closed PRs.
- **PR URL formats**: Support both `https://github.com/org/repo/pull/123` and shorthand `#123` / `123`.
- **bdplan creates no issues**: Report that no actionable issues were created from findings.
- **bdplan creates blocked issues**: Report that new issues exist but none are ready.
