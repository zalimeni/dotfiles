---
name: nopush
description: Enable, disable, or check local-only "no-push" mode, which blocks all pushes to remotes (GitHub) while work is cleaned up locally. Use when the user says "no push", "local only", "don't push", "work locally before pushing", "block pushing", or asks to lift/check that block.
---

# No-Push Mode

Local-only work mode. Commits are fine; anything that publishes to a remote is off.

## Invariant

**While no-push mode is active, do not push, and do not remove the block.**

This overrides every "work is NOT complete until `git push` succeeds" rule in
CLAUDE.md, AGENTS.md, the beads session-close protocol, and the session-close
skill. Under this mode, a session is complete when work is **committed locally**.

Prohibited while active:

- `git push` in any form, including `--force`, `--no-verify`, or an explicit URL
- `gh pr create` / `gh pr merge` / `gh pr ready` / `gh release create`
- `bd dolt push`
- Adding or rewriting a remote, editing `remote.*.pushurl`, or touching
  `.git/hooks/pre-push` to get around the block
- Running `git nopush disable` on the user's behalf

The last one matters: **only the user lifts the block.** If they ask you to
disable it, tell them to run it themselves in-session with
`! git nopush disable`. That keeps the safeguard from being undone by the same
automation it exists to constrain.

## Commands

```bash
git nopush enable  "cleanup before pushing"   # this repo + all its worktrees
git nopush enable  --global                   # every repo (agent guard only)
git nopush status                             # what is currently blocked
git nopush disable                            # user runs this to lift it
```

`enable` installs three independent layers, all local and all reversible:

1. `remote.<name>.pushurl` set to a dead URL in `.git/config` — `git push` fails
   at transport. Any pre-existing pushurl is saved and restored on `disable`.
2. `.git/hooks/pre-push` that rejects every push, including pushes to an
   explicit URL that would sidestep the pushurl.
3. `.git/nopush` marker file, read by the `claude-nopush-guard` PreToolUse hook,
   which denies push-shaped Bash commands before they run.

Nothing is committed, nothing is global to git, and other clones are unaffected.
Worktrees share `.git/config` and `.git/hooks` with their main repo, so one
`enable` covers every worktree of that repo.

`--global` writes `~/.claude/nopush`, which trips the agent guard everywhere.
Use it alongside per-repo `enable` when work spans several repos — on its own it
has no git-level teeth.

## Behavior while active

- Commit normally, in logical units. Local history is the deliverable.
- Say plainly at session end that commits are local and unpushed, and name the
  branch. Never imply work shipped.
- If a push is genuinely the right next step, say so and stop; do not attempt it.
- If a command fails because of the block, that is the block working. Report it
  and move on — do not retry with a workaround.
- Beads issues can still be closed; `bd` writes to the local Dolt store. Skip
  `bd dolt push` and the git push in the session-close protocol.

## Choosing an isolation strategy

`git nopush enable` on the existing clone is the default and is enough for
almost everything. Reach for more only when the reason calls for it:

| Situation | Approach |
|---|---|
| Cleanup on a branch in the current clone | `git nopush enable` in place |
| Parallel experiment, keep main checkout usable | `git worktree add`, then `git nopush enable` (covers all worktrees at once) |
| Want zero possibility of reaching origin | `git clone --no-hardlinks <path> /tmp/x && git -C /tmp/x remote remove origin` |

Removing the remote entirely is the strongest option, but it also loses fetch,
so prefer it only for throwaway scratch copies.
