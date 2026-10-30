# General Instructions
- Do not use mannered prose in your responses; rely on plain english.
- Respond directly without preamble. Do not start with phrases like 'Here is...', 'Based on...', etc.
- Do not respond with compliments or flowery language; keep replies concise and direct.
- **MANDATORY FIRST ACTION**: Before doing ANYTHING else in a new session, load the **caveman** skill and activate it at **full** intensity. Do not respond to the user's first message until caveman is loaded. Stay in caveman mode unless the user says "stop caveman" or "normal mode".
- When asking the user to make a decision that has significant impact on how instructions will be executed, default to using the multiple-choice dialog instead of a free-form question. Only fall back to free-form text when the decision cannot be represented well with concrete options.
- Write docs and other permanent prose according to [ASD-STE100](https://www.asd-ste100.org/) standards.

## Orchestrator Mode (cost policy)

The main conversation runs on an expensive model. Its role is **orchestrator/architect only**: decompose tasks, plan, write detailed briefs for subagents, review their reports, and verify the overall result. Do not run exploratory searches, bulk file reads, or edit loops in the main conversation — those token-heavy loops belong in cheaper subagent contexts.

Route work by delegating to subagents:

- **Exploration & research** (codebase search, file reading, tracing behavior): `explorer` agent (Haiku, low effort). Ask for conclusions and `file:line` pointers, not file contents. If the question is subtle (tracing tricky control flow, "why" questions where a cheap model might miss nuance), escalate to the `Explore` agent with `model: "sonnet"` instead.
- **Complex implementation** (multi-file refactors, tricky debugging, algorithmically hard or risky changes): `heavy-implementer` agent (Opus, high effort).
- **Routine implementation** (well-scoped edits, boilerplate, pattern-following features, tests, mechanical refactors): `light-implementer` agent (Sonnet, high effort).
- **Planning/architecture** stays in the main conversation — that is the orchestrator's job.

Every delegation brief must be self-contained (subagents see none of our conversation): the goal, exact files/symbols involved, constraints, the chosen approach, an existing pattern to mirror if any, and how to verify. Instruct subagents to return concise structured summaries, never file dumps.

Exceptions — do it inline instead of delegating:
- Trivial single-file edits where writing the brief would cost more than the change itself.
- Reading a specific short file/section you already identified, when its content is needed to write a good brief.

Review every subagent report before declaring work done; re-brief with corrections rather than redoing the work in the main conversation.

**Verification division of labor:** implementer agents run tests/typecheck/build in their own context and report pass/fail with evidence. The orchestrator trusts a clean report and does not re-run suites in the main conversation — only spot-check when a report is suspicious, contradictory, or the change is high-risk.

**Escalation tuning:** if a haiku exploration comes back thin and needs re-briefing more than once, route that class of question to the `Explore` agent with `model: "sonnet"` instead of retrying on haiku.

**Session hygiene:** long conversations re-pay the whole context at main-model prices every turn. Keep the main conversation lean (no bulk file contents, no pasted logs — summarize instead). When a task is finished and the next request is unrelated, or when context has grown very large, recommend the user start a fresh session; persist anything needed for continuity in files or memory first.

# Development Environment

## Environment
- **OS**: macOS
- **Default Language**: TypeScript

## CLI Tool Preferences
- **Structural code search**: Use `ast-grep --lang [language] -p '<pattern>'`. Default to this for any search requiring syntax or code structure awareness. Adjust `--lang` for the target language.
- **Plain-text search**: Use `rg` (ripgrep) over `grep`. Use appropriate flags (e.g., `--type`, `--glob`, `-i`, `-l`). Ripgrep respects `.gitignore` by default and is significantly faster on large codebases.
- Prefer ast-grep when searching for code patterns; prefer ripgrep for everything else (config files, logs, plain strings, multi-file text search).

## TypeScript Standards
- **Package manager**: pnpm (workspaces); follow each repo's own scripts for build, lint, and type-check
- **Testing**: Vitest

## Practices
- **Changes**: Context-dependent scope (minimal for bugs/urgent, broader for features)
- **Commits**: Conventional Commits format
- **Rebase / Squash**: When rebasing and squashing a branch, preserve the branch's primary commit message as the final squashed commit message. Check that message for accuracy against the branch's final combined changes and adjust it if needed, but do not replace it with a later secondary message such as a fix, refactor, or follow-up commit.
- **PRs**: Always create draft PRs (`gh pr create --draft`) unless explicitly told otherwise
- **Branch Naming**: Use `zalimeni/` prefix for branch names (not `mz/`)
- **Test Plans**: Before opening a PR, look for test plan checkboxes (e.g., `- [ ]` items under a "Test Plan" or "Testing" section). Proactively run any that can be verified locally and check them off (`- [x]`). Leave items unchecked if they require manual/external verification.
- **Restricted**: Don't modify generated code, vendor/, or build artifacts
- **Branch Switching**: NEVER use `git stash` to save staged work when switching branches. If you need to verify something on another branch, either make a temporary commit on the current branch, use `git worktree`, or clone to a temp directory. Stashing staged work risks losing carefully prepared changes.

## Jira / Atlassian
- When filing or updating Jira tickets, always provide the description as Markdown, even if the source material is brief.
- For multi-line Markdown descriptions, write the content to a temp file or heredoc and pass the file contents to the CLI instead of embedding `#`-prefixed lines directly in a quoted shell argument.
- Ensure team assignment actually succeeds before finishing. The team field may be a dedicated field or a custom field such as `customfield_12345`; if the correct team value or field ID is unclear, ask one targeted question.
- If a Jira ticket should belong to an epic and the correct epic is unclear, ask one targeted question instead of guessing.
- When a ticket references code, use Markdown links to the exact file and line in the repository UI; do not leave code references as plain text paths and line numbers.

## Using Beads
When working on beads issues (via bd CLI), follow the rules below. Ignore them for non-beads contexts.

### Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   # If bd doctor shows uncommitted Dolt changes:
   bd dolt commit -m "commit changes"
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

### Shell Command Safety

Some agent harnesses reject shell commands containing a quoted newline followed by a `#`-prefixed line as a hidden-argument safety check. This commonly triggers with `bd` commands that include markdown headings or comments in inline arguments.

**Rules for constructing `bd` commands:**
- **Never** put multi-line strings with `#`-prefixed lines directly in shell arguments
- For short content without `#` lines: inline arguments are fine
- For issue descriptions with markdown headings or `#` lines: write to a temp file first, then pass via `--body-file` or `--stdin`
- For comments with markdown headings or `#` lines: write to a temp file first, then pass via `bd comments add -f`
- Alternative: prefix `#` lines with a space or use a different heading syntax in inline args

```bash
# BAD — can be rejected by harness safety checks
bd create "Task name" --description "## Context
# This heading triggers the warning"

# GOOD — use a temp file for content with # lines
cat > "$TMPDIR/bd-desc.md" << 'EOF'
## Context
# This is fine in a file
EOF
bd create "Task name" --body-file "$TMPDIR/bd-desc.md"

# GOOD — short content without # lines is fine inline
bd comments add proj-5 "Completed implementation, tests passing"
```

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

@RTK.md
