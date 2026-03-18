# Dotfiles

Personal dotfiles for macOS. Shell functions, aliases, git config, terminal setup, and helper scripts.

## Structure

```
bat/              bat config + everforest theme
bin/              custom scripts (tmux-sessionizer, git helpers, etc.)
cmd/              Go CLI commands (sp2md)
git/              git config overrides (hashi/.gitconfig)
internal/         Go library code
system/           shell aliases (.alias) and functions (.function)
wallpaper/        desktop wallpapers
everforest_colors terminal color palette reference

.config/
  claude/         Claude Code config (CLAUDE.md, settings.json, skills, agents)
  opencode/       OpenCode config (opencode.json, generated AGENTS.md, translated agents)
  ghostty/        Ghostty terminal config
  k9s/            Kubernetes TUI config
  mise/           mise runtime/tool version manager
  ranger/         ranger file manager
  tmux/           tmux config
```

## Setup

> **WARNING:** Before installing, verify you're ok with the files that will be overwritten, or save a backup!

Clone the repo and run the installer:

```sh
git clone git@github.com:zalimeni/dotfiles.git ~/dotfiles
cd ~/dotfiles && ./install.sh
```

The installer detects macOS vs Linux and sets up Claude Code config, OpenCode config,
git identity, shell config, etc. It's idempotent — safe to re-run.

For local work sessions, `system/.work_env` can be used. It's git-ignored but acts the same as `system/.env`. Similar analogies exist for `system/.alias` and `system/.function`

For Claude Code online sandboxes, see [Sandbox setup](#sandbox-setup).

### Claude vs. OpenCode

The installer manages subtle config, frontmatter, and other differences between these files by preferring a symlink to the Claude version in this repo, but actively translating when a clone isn't sufficient or compatible. For example, the OpenCode configured agents differ from Claude Code, and assume sourcing from GitHub Copilot (what I use at work).

### Sandbox setup

In the Claude Environments UI, set:

- **Setup command**: `git clone https://github.com/zalimeni/dotfiles.git ~/dotfiles && ~/dotfiles/install.sh`
- **Environment variables**: `CONTEXT7_KEY`, plus any secrets from `system/.private_env`

Use `tmux-agent-worktree` for new delegated agent worktrees; harness selection comes
from `AGENT_HARNESS` instead of a Claude-specific launcher name.

## Tools

- **Shell**: zsh + oh-my-zsh
- **Terminal**: Ghostty + tmux
- **Editor**: neovim
- **Go**: go, golangci-lint, staticcheck
- **Utilities**: bat, eza, fzf, ranger, ripgrep, fd, delta
- **Infra**: k9s, mise, gh

## Credits

The [dotfiles community](https://dotfiles.github.io)
