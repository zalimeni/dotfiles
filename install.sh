#!/usr/bin/env bash
#
# Dotfiles installer — works on macOS (primary) and Linux (Claude Code sandbox).
# Idempotent: safe to re-run.
#
# Usage:
#   git clone https://github.com/zalimeni/dotfiles.git ~/dotfiles
#   cd ~/dotfiles && ./install.sh
#
set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"
PLATFORM="$(uname -s)"

info()  { printf '  [ \033[00;34m..\033[0m ] %s\n' "$1"; }
ok()    { printf '  [ \033[00;32mOK\033[0m ] %s\n' "$1"; }
warn()  { printf '  [ \033[0;33m!!\033[0m ] %s\n' "$1"; }
fail()  { printf '  [\033[0;31mFAIL\033[0m] %s\n' "$1"; exit 1; }

# --- helpers ----------------------------------------------------------------

link_file() {
  local src="$1" dst="$2"
  mkdir -p "$(dirname "$dst")"

  if [ -L "$dst" ]; then
    rm "$dst"
  elif [ -f "$dst" ] || [ -d "$dst" ]; then
    mv "$dst" "${dst}.backup"
    warn "backed up existing $dst to ${dst}.backup"
  fi

  ln -sf "$src" "$dst"
  ok "linked $dst -> $src"
}

generate_agent_config() {
  local generator="$DOTFILES_DIR/bin/sync-agent-config"

  [ -x "$generator" ] || fail "missing generator: $generator"

  info "regenerating translated agent config"
  "$generator"
}

# --- opencode config --------------------------------------------------------

install_opencode() {
  info "installing opencode config"

  local opencode_dir="$HOME/.config/opencode"
  mkdir -p "$opencode_dir"

  # AGENTS.md — generated from the canonical Claude instructions.
  link_file "$DOTFILES_DIR/.config/opencode/AGENTS.md" "$opencode_dir/AGENTS.md"

  # opencode.json — main config (model, MCP servers).
  # Does NOT contain auth; opencode stores credentials in ~/.local/share/opencode/.
  link_file "$DOTFILES_DIR/.config/opencode/opencode.json" "$opencode_dir/opencode.json"

  # package.json — plugin dependencies for ~/.config/opencode/plugins/.
  # OpenCode installs these with Bun at startup for local plugins like peon-ping.
  if [ -f "$DOTFILES_DIR/.config/opencode/package.json" ]; then
    link_file "$DOTFILES_DIR/.config/opencode/package.json" "$opencode_dir/package.json"
  fi
  # Agents — per-agent symlinks to the generated opencode translations.
  # These are generated from .config/claude/agents/ by bin/sync-agent-config.
  # Skills are NOT linked here: opencode natively reads ~/.claude/skills/, which
  # is already set up by install_claude().
  if [ -d "$DOTFILES_DIR/.config/opencode/agents" ]; then
    if [ -L "$opencode_dir/agents" ]; then
      rm "$opencode_dir/agents"
    elif [ -f "$opencode_dir/agents" ]; then
      mv "$opencode_dir/agents" "$opencode_dir/agents.backup"
      warn "backed up existing $opencode_dir/agents to $opencode_dir/agents.backup"
    fi
    mkdir -p "$opencode_dir/agents"
    for agent_file in "$DOTFILES_DIR/.config/opencode/agents"/*.md; do
      link_file "$agent_file" "$opencode_dir/agents/${agent_file##*/}"
    done
  fi

  # Pre-commit hook — keeps opencode agents in sync when Claude agents change.
  local hooks_dir="$DOTFILES_DIR/.git/hooks"
  if [ -d "$hooks_dir" ]; then
    link_file "$DOTFILES_DIR/bin/pre-commit-sync-agents" "$hooks_dir/pre-commit"
    ok "installed pre-commit hook (agent sync)"
  else
    warn "no .git/hooks directory found — skipping pre-commit hook install"
  fi

  echo ""
  info "opencode post-install steps:"
  info "  1. cd ~/.config/opencode && npm install  (installs plugins)"
  info "  2. Run 'opencode /connect', select 'GitHub Copilot', complete auth"
  info "  3. Run '/models' inside opencode to confirm model IDs match opencode.json"
  info "  4. Set CONTEXT7_KEY env var (or Claude Environments UI) for context7 MCP"
}

# --- Remote skills ----------------------------------------------------------

install_remote_skills() {
  local manifest="$DOTFILES_DIR/.config/claude/skills/remote-skills.manifest"
  if [ ! -f "$manifest" ]; then
    warn "no remote-skills.manifest found, skipping remote skill downloads"
    return
  fi

  info "downloading remote skills"

  while IFS= read -r line || [ -n "$line" ]; do
    # Skip blanks and comments
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue

    local skill_name repo repo_path
    read -r skill_name repo repo_path <<< "$line"

    local dest="$HOME/.claude/skills/$skill_name"

    # Download SKILL.md (and any sibling files) from the GitHub repo.
    # We use the GitHub API to list directory contents, then curl each file.
    local api_url="https://api.github.com/repos/$repo/contents/$repo_path?ref=main"
    local file_list
    file_list="$(curl -fsSL "$api_url" 2>/dev/null)" || {
      warn "failed to fetch file list for skill '$skill_name' from $repo"
      continue
    }

    # Remove stale symlink (e.g. from previous install that symlinked into repo)
    [ -L "$dest" ] && rm "$dest"
    mkdir -p "$dest"

    # Parse JSON array — each element has "name" and "download_url".
    # Use grep+sed to avoid a hard jq dependency (jq may not exist on
    # sandbox Linux images).
    local names urls
    names="$(echo "$file_list" | grep '"name"' | sed 's/.*"name": *"\([^"]*\)".*/\1/')"
    urls="$(echo "$file_list" | grep '"download_url"' | sed 's/.*"download_url": *"\([^"]*\)".*/\1/')"

    # Walk both lists in lockstep
    paste <(echo "$names") <(echo "$urls") | while IFS=$'\t' read -r fname furl; do
      [ -z "$fname" ] && continue
      curl -fsSL "$furl" -o "$dest/$fname" 2>/dev/null || {
        warn "failed to download $fname for skill '$skill_name'"
        continue
      }
    done

    ok "downloaded skill '$skill_name' from $repo"
  done < "$manifest"
}

# --- Claude Code config -----------------------------------------------------

install_claude() {
  info "installing Claude Code config"

  local claude_dir="$HOME/.claude"
  mkdir -p "$claude_dir"

  # Canonical instruction file — exposed under both Claude-compatible and
  # tooling-agnostic names so Claude Code and opencode both find it.
  link_file "$DOTFILES_DIR/.config/claude/CLAUDE.md" "$claude_dir/CLAUDE.md"
  link_file "$DOTFILES_DIR/.config/claude/CLAUDE.md" "$claude_dir/AGENTS.md"

  # config.json (MCP servers, LSP)
  link_file "$DOTFILES_DIR/.config/claude/config.json" "$claude_dir/config.json"

  # Skills — per-skill symlinks (skip remote skills already in ~/.claude/skills/)
  if [ -d "$DOTFILES_DIR/.config/claude/skills" ]; then
    mkdir -p "$claude_dir/skills"
    # Build set of remote skill names to skip
    local manifest="$DOTFILES_DIR/.config/claude/skills/remote-skills.manifest"
    local -A remote_skills=()
    if [ -f "$manifest" ]; then
      while IFS= read -r _line || [ -n "$_line" ]; do
        [[ -z "$_line" || "$_line" =~ ^[[:space:]]*# ]] && continue
        local _rname; read -r _rname _ _ <<< "$_line"
        remote_skills["$_rname"]=1
      done < "$manifest"
    fi
    for skill_dir in "$DOTFILES_DIR/.config/claude/skills"/*/; do
      [ -d "$skill_dir" ] || continue
      local skill_name
      skill_name="$(basename "$skill_dir")"
      # Skip remote skills — they're downloaded directly to ~/.claude/skills/
      [[ -n "${remote_skills[$skill_name]+x}" ]] && continue
      link_file "$skill_dir" "$claude_dir/skills/$skill_name"
    done
    ok "linked skills"
  fi

  # Agents
  if [ -d "$DOTFILES_DIR/.config/claude/agents" ]; then
    link_file "$DOTFILES_DIR/.config/claude/agents" "$claude_dir/agents"
  fi

  # settings.json — platform-dependent
  if [ "$PLATFORM" = "Darwin" ]; then
    link_file "$DOTFILES_DIR/.config/claude/settings.json" "$claude_dir/settings.json"
  else
    install_claude_sandbox_settings "$claude_dir/settings.json"
  fi
}

install_claude_sandbox_settings() {
  local dst="$1"
  local src="$DOTFILES_DIR/.config/claude/settings.json"
  info "generating sandbox settings.json from $src via jq"

  if ! command -v jq &>/dev/null; then
    fail "jq is required to generate sandbox settings.json"
  fi

  # Filter the canonical settings.json:
  #   - Remove env (secrets/work-specific — set in Claude Environments UI)
  #   - Remove hooks (peon-ping, macOS audio — no use in sandbox)
  #   - Remove sandbox block (macOS-specific paths)
  #   - Remove skipDangerousModePermissionPrompt (local-only preference)
  #   - Strip permissions.allow entries containing absolute paths (macOS home dir)
  jq '
    del(.env, .hooks, .sandbox, .skipDangerousModePermissionPrompt)
    | .permissions.allow |= map(select(test("/") | not))
  ' "$src" > "$dst"

  ok "wrote $dst (filtered for sandbox — no hooks, no macOS paths)"
}

# --- Git identity -----------------------------------------------------------

install_git() {
  info "configuring git identity"

  # Only set if not already configured (don't clobber existing global config)
  if [ -z "$(git config --global user.name 2>/dev/null || true)" ]; then
    git config --global user.name "Michael Zalimeni"
    ok "set git user.name"
  else
    ok "git user.name already set: $(git config --global user.name)"
  fi

  if [ -z "$(git config --global user.email 2>/dev/null || true)" ]; then
    git config --global user.email "mzalimeni@gmail.com"
    ok "set git user.email (personal default)"
  else
    ok "git user.email already set: $(git config --global user.email)"
  fi

  # includeIf for HashiCorp repos
  local hashi_cfg="$DOTFILES_DIR/git/hashi/.gitconfig"
  if [ -f "$hashi_cfg" ]; then
    git config --global --replace-all \
      "includeIf.hasconfig:remote.*.url:git@github.com:hashicorp/**.path" \
      "$hashi_cfg"
    ok "set includeIf for hashicorp org -> $hashi_cfg"
  fi

  # git helpers
  if [ -f "$DOTFILES_DIR/git/.githelpers" ]; then
    git config --global alias.lg "!source $DOTFILES_DIR/git/.githelpers && pretty_git_log"
    git config --global alias.brv "!source $DOTFILES_DIR/git/.githelpers && pretty_git_branch_sorted"
    ok "set git aliases (lg, brv)"
  fi
}

# --- Shell config -----------------------------------------------------------

install_shell() {
  info "installing shell config"

  # Add dotfiles/bin to PATH via profile
  local profile="$HOME/.profile"
  local path_line="export PATH=\"$DOTFILES_DIR/bin:\$PATH\""

  if [ -f "$profile" ] && grep -qF "$DOTFILES_DIR/bin" "$profile"; then
    ok "PATH already includes dotfiles/bin"
  else
    echo "" >> "$profile"
    echo "# Dotfiles" >> "$profile"
    echo "$path_line" >> "$profile"
    ok "added dotfiles/bin to PATH in $profile"
  fi

  # Source portable aliases and functions in shell rc.
  # On macOS this is handled by zsh/.zshrc_custom — skip.
  if [ "$PLATFORM" != "Darwin" ]; then
    local rc="$HOME/.bashrc"
    [ -f "$HOME/.zshrc" ] && rc="$HOME/.zshrc"

    local source_block="# Dotfiles shell config
for f in \"$DOTFILES_DIR\"/system/.{env.sandbox,alias,function}; do
  [ -f \"\$f\" ] && source \"\$f\"
done"

    if grep -qF "Dotfiles shell config" "$rc" 2>/dev/null; then
      ok "shell config already sourced in $rc"
    else
      echo "" >> "$rc"
      echo "$source_block" >> "$rc"
      ok "added shell config to $rc"
    fi
  fi
}

# --- Sandbox env file -------------------------------------------------------

install_sandbox_env() {
  local env_file="$DOTFILES_DIR/system/.env.sandbox"
  if [ -f "$env_file" ]; then
    ok "sandbox env already exists"
    return
  fi

  info "creating system/.env.sandbox (portable env vars)"
  cat > "$env_file" << 'ENV'
# Portable environment variables for Linux/sandbox — sourced by install.sh
# macOS uses system/.env instead (has Homebrew, platform-specific paths)

export EDITOR=vim
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"
export HUSKY=0
export HUSKY_SKIP_HOOKS=1
ENV
  ok "wrote $env_file"
}

# --- bat config (optional) --------------------------------------------------

install_bat() {
  if ! command -v bat &>/dev/null && ! command -v batcat &>/dev/null; then
    warn "bat not found, skipping bat config"
    return
  fi

  link_file "$DOTFILES_DIR/bat" "$HOME/.config/bat"

  if command -v bat &>/dev/null; then
    bat cache --build 2>/dev/null && ok "rebuilt bat cache" || true
  elif command -v batcat &>/dev/null; then
    batcat cache --build 2>/dev/null && ok "rebuilt bat cache (batcat)" || true
  fi
}

# --- main -------------------------------------------------------------------

main() {
  echo ""
  echo "  dotfiles installer — $([ "$PLATFORM" = "Darwin" ] && echo "macOS" || echo "Linux/sandbox")"
  echo ""

  generate_agent_config
  install_remote_skills
  install_opencode
  install_claude
  install_git
  install_shell

  if [ "$PLATFORM" != "Darwin" ]; then
    install_sandbox_env
  fi

  install_bat

  echo ""
  ok "all done"

  if [ "$PLATFORM" != "Darwin" ]; then
    echo ""
    warn "Sandbox reminder: set these in the Claude Environments UI:"
    warn "  - CONTEXT7_KEY (for context7 MCP server)"
    warn "  - Any secrets from system/.private_env"
    warn "  - SSH key or GitHub token for private repo access"
    echo ""
  fi
}

main "$@"
