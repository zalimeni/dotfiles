# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Session Completion

Work is NOT complete until `git push` succeeds. Follow the session-close protocol:
`bd sync && git pull --rebase && git push`

## Pull Requests

When creating or updating a PR as the agent:

- Ensure PR attribution is present either by relying on the repo's PR template and updating the PR afterward, or by including it directly in any custom PR body
- If you write a custom PR description, end it with a final attribution line in this format: `🤖 Generated with OpenCode (<exact model>)`
- If you create a git commit, include commit-body attribution plus a co-author trailer using the exact model, for example:
  - `Generated with OpenCode (gpt-5.4).`
  - `Co-authored-by: OpenCode (gpt-5.4) <noreply@openai.com>`
