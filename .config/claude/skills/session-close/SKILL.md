---
name: session-close
description: Protocol for properly ending a coding session - ensures all work is committed, pushed, and handed off correctly.
license: MIT
metadata:
  category: workflow
  tools: git, bd
---

## When to use me

Use this skill when ending a work session. This ensures all work is properly saved, pushed, and documented for the next session.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git fetch origin
   git rebase origin/main
   bd sync
   # Push with upstream tracking (handles both new and existing branches)
   git push -u origin HEAD
   git status  # Verify push succeeded
   ```
   If `git push` fails with "no upstream branch", the `-u` flag handles it.
   If it fails with conflicts after rebase, resolve them and retry.
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

## Critical Rules

- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

## Checklist

Before saying "done" or "complete", run this checklist:

```
[ ] 1. git status                 (check what changed)
[ ] 2. git add && git commit      (stage and commit changes)
[ ] 3. bd sync                    (sync beads changes)
[ ] 4. git push -u origin HEAD    (push to remote, set upstream)
[ ] 5. git status                 (verify clean working tree)
```

**NEVER skip this.** Work is not done until pushed.
