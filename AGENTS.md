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

## Pull Requests

When creating or updating a PR as the agent:

Every attribution links to the AI attribution policy, so readers can see why it is there.
Get `<permalink>` by running `ai-attribution-link` (in `bin/`). It prints a commit-pinned
permalink to the latest `AI_ATTRIBUTION.md` on `main` and refreshes its cache once a day.
Never write the URL as a bare link in Markdown; always use the anchor text shown below.

- Ensure PR attribution is present either by relying on the repo's PR template and updating the PR afterward, or by including it directly in any custom PR body
- If you write a custom PR description, end it with a final attribution line in this format: `🤖 [Authored using AI](<permalink>) (<exact model>)`
- If you write PR review comment replies, end each with a blank line then attribution in this format: `🤖 _[Authored by AI](<permalink>) (<exact model>)_`
- If you create a git commit, include commit-body attribution with the policy link plus a co-author trailer using the exact model. Git does not render Markdown, so commits use the plain permalink, for example:
  - `Generated with OpenCode (claude-opus-4.6).`
  - `AI attribution policy: <permalink>`
  - `Co-authored-by: OpenCode (claude-opus-4.6) <noreply@anthropic.com>`
