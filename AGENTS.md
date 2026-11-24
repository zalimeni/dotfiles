# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd prime` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd dolt commit -m "..." # Commit pending beads Dolt changes when needed
```

## Session Completion

Unless explicitly instructed not to push changes, work is NOT complete until `git push` succeeds. Follow the session-close protocol:
`git pull --rebase && git push`

If `bd doctor` reports uncommitted Dolt changes, commit them first with
`bd dolt commit -m "..."`.
