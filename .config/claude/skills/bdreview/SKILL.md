---
name: bdreview
description: review code changes since a baseline commit and plan fixes
argument-hint: "[target] [label]"
allowed-tools: Skill, Task, Bash(bd:*), Bash(git *), Bash(gh *)
---

# BD Review

## Overview

Review code changes for a given target (commit, range, or PR), categorize findings by severity, and create fix issues via `/bdplan` for actionable findings. Can be used standalone after manual coding, to review a PR, or invoked as part of `/bdloop`.

## Arguments

$ARGUMENTS

- `target` (optional, default: `main`): What to review. One of:
  - **Refname** — branch or tag name; diffs `refname..HEAD` (e.g., `main`, `origin/main`, `v1.2.0`)
  - **Single SHA** — diffs `SHA..HEAD` (e.g., `abc1234`)
  - **Ref range** — diffs the range as-is (e.g., `main..feature`, `abc1234..def5678`)
  - **PR reference** — diffs the pull request (e.g., `#123`, `123`, or a GitHub PR URL)
- `label` (optional): Context label for output cards (e.g., "Iteration 2")

## Instructions

### 1. Resolve Target

If no target is provided, default to `main`.

Detect the input type and resolve it into `DIFF_CMD` and `LOG_CMD` variables used by later steps.

| Input | Detection | DIFF_CMD | LOG_CMD |
| ----- | --------- | -------- | ------- |
| PR ref (`#N`, bare number, or `github.com/.../pull/N` URL) | matches `#?\d+$` or GH PR URL | `gh pr diff N` | `gh pr view N` |
| Ref range (`A..B`) | contains `..` | `git diff A..B` | `git log A..B --oneline` |
| Single branch/tag ref | resolves to branch or tag name | `git diff REF...HEAD` | `git log $(git merge-base REF HEAD)..HEAD --oneline` |
| Single SHA | resolves to a commit SHA | `git diff SHA..HEAD` | `git log SHA..HEAD --oneline` |

**Sync base ref before diffing (non-PR targets only):**

When the target resolves to a local branch name (not a bare SHA, not a PR), the local ref may be behind the remote. Stale local refs cause the review to include already-merged commits or miss recently-pushed work.

```bash
# Fetch the latest state of the base ref from origin
git fetch origin <branch>:<branch>
```

- If the current branch IS the base branch (e.g., reviewing `main` while on `main`), skip the fetch-with-update (it would fail on a checked-out branch). Instead, run `git pull --ff-only` to advance the local branch.
- If fetch fails (e.g., ref doesn't exist on remote, is a checked-out branch, or is a tag/SHA), fall through — the ref may be local-only, which is fine.
- For ref ranges (`A..B`), apply the same logic to both A and B if they look like branch names.

**Use actual branch commits, not just tree differences, when reviewing against a base branch:**

- For a single branch/tag base such as `main`, always compute `MERGE_BASE=$(git merge-base REF HEAD)` after syncing the ref.
- Review the branch's actual work with `git diff REF...HEAD` and `git log $MERGE_BASE..HEAD --oneline`.
- Build a changed-file list from `git diff --name-only REF...HEAD` and tell the review agent to focus on those files.
- Do **not** report findings from files that are different only because the base branch moved forward after the feature branch diverged.
- If a shared file changed on the branch (for example shared gateway wiring used by a new API), it is still in scope even if the branch's primary feature is narrower.

**Validate the resolved target:**

- **PR**: `gh pr view N --json number` must succeed. If the PR is not found, exit with an error.
- **Ref range**: `git rev-parse A` and `git rev-parse B` must both resolve to commits.
- **Single ref**: `git rev-parse REF` must resolve to a commit.

If validation fails, exit with an error message.

### 2. Check for Changes

Run `LOG_CMD` to verify there are changes to review.

- **PR**: `gh pr view N --json additions,deletions` — if both are 0, no changes.
- **Ref range / single ref**: if the log output is empty, no changes.

For single branch/tag refs, also use `git diff --name-only REF...HEAD` to confirm there are files changed on the branch. If the file list is empty, there is nothing to review.

If no changes, output a message and exit — nothing to review.

```
┌─ REVIEW RESULT [label] ────────────────
│ No changes found for [target].
│ Verdict: SKIP
└─────────────────────────────────────────
```

### 3. Run Review Agent

Invoke the review agent scoped to the resolved target:

**For ref-based targets (single ref or range):**

```
Task(
  description="Review changes for [target]",
  subagent_type="review",
  prompt="Review the code changes for [target].

Use this command to see the diff:
  [DIFF_CMD]

Use this command to see the commit log:
  [LOG_CMD]

Use this command to see the changed files that are actually in scope:
  [FILES_CMD]

Review ONLY files returned by [FILES_CMD]. Ignore files that differ only because the base branch advanced after the branch diverged.

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

Before categorizing, discard any finding that is out of scope for the resolved target, including findings from files not present in `[FILES_CMD]` for branch/tag reviews.

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
- **No target provided**: Default to `main` as the base ref.
- **Invalid target**: Exit with error (Step 1) — ref doesn't exist, PR not found, etc.
- **Base branch moved ahead**: For branch/tag reviews, use three-dot diff and merge-base-scoped commit log so the review stays limited to actual branch commits.
- **Closed/merged PR**: Still reviewable — `gh pr diff` works on closed PRs.
- **PR URL formats**: Support both `https://github.com/org/repo/pull/123` and shorthand `#123` / `123`.
- **bdplan creates no issues**: Report that no actionable issues were created from findings.
- **bdplan creates blocked issues**: Report that new issues exist but none are ready.
