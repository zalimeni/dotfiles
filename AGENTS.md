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

Work is NOT complete until `git push` succeeds. Follow the session-close protocol:
`git pull --rebase && git push`

If `bd doctor` reports uncommitted Dolt changes, commit them first with
`bd dolt commit -m "..."`.

## Pull Requests

When creating or updating a PR as the agent:

- Ensure PR attribution is present either by relying on the repo's PR template and updating the PR afterward, or by including it directly in any custom PR body
- If you write a custom PR description, end it with a final attribution line in this format: `🤖 Generated with OpenCode (<exact model>)`
- If you write PR review comment replies, end each with a blank line then attribution in this format: `🤖 _Generated with OpenCode (<exact model>)_`
- If you create a git commit, include commit-body attribution plus a co-author trailer using the exact model, for example:
  - `Generated with OpenCode (claude-opus-4.6).`
  - `Co-authored-by: OpenCode (claude-opus-4.6) <noreply@anthropic.com>`
