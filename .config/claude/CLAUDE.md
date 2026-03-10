# General Instructions
- Respond directly without preamble. Do not start with phrases like 'Here is...', 'Based on...', etc.
- Do not respond with compliments or flowery language; keep replies concise and direct.

# Development Environment

## Environment
- **OS**: macOS
- **Default Language**: Go

## CLI Tool Preferences
- **Structural code search**: Use `ast-grep --lang [language] -p '<pattern>'`. Default to this for any search requiring syntax or code structure awareness. Adjust `--lang` for the target language.
- **Plain-text search**: Use `rg` (ripgrep) over `grep`. Use appropriate flags (e.g., `--type`, `--glob`, `-i`, `-l`). Ripgrep respects `.gitignore` by default and is significantly faster on large codebases.
- Prefer ast-grep when searching for code patterns; prefer ripgrep for everything else (config files, logs, plain strings, multi-file text search).

## Go Standards
- **Build**: Make/Makefiles
- **Testing**: Testify (testify/assert, testify/mock, testify/suite)
- **Linting**: golangci-lint, staticcheck
- **Docs**: Godoc comments on all exported APIs; internal code only where non-obvious

## Practices
- **Changes**: Context-dependent scope (minimal for bugs/urgent, broader for features)
- **Commits**: Conventional Commits format
- **PRs**: Always create draft PRs (`gh pr create --draft`) unless explicitly told otherwise
- **Restricted**: Don't modify generated code, vendor/, or build artifacts

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
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

### Shell Command Safety

Claude Code rejects shell commands containing a quoted newline followed by a `#`-prefixed line (security check against hidden arguments). This commonly triggers with `bd` commands that include markdown headings or comments in inline arguments.

**Rules for constructing `bd` commands:**
- **Never** put multi-line strings with `#`-prefixed lines directly in shell arguments
- For short content without `#` lines: inline arguments are fine
- For content with markdown headings or `#` lines: write to a temp file first, then pass via `--file` or stdin
- Alternative: prefix `#` lines with a space or use a different heading syntax in inline args

```bash
# BAD — will be rejected by Claude Code permission check
bd create "Task name" --description "## Context
# This heading triggers the warning"

# GOOD — use a temp file for content with # lines
cat > "$TMPDIR/bd-desc.md" << 'EOF'
## Context
# This is fine in a file
EOF
bd create "Task name" --description "$(cat "$TMPDIR/bd-desc.md")"

# GOOD — short content without # lines is fine inline
bd comment proj-5 "Completed implementation, tests passing"
```

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
