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
  opencode/       OpenCode config (opencode.json; reuses Claude agents and instructions)
  ghostty/        Ghostty terminal config
  k9s/            Kubernetes TUI config
  mise/           mise runtime/tool version manager
  ranger/         ranger file manager
  tmux/           tmux config
```

## Setup

Clone the repo and run the installer:

```sh
git clone git@github.com:zalimeni/dotfiles.git ~/dotfiles
cd ~/dotfiles && ./install.sh
```

The installer detects macOS vs Linux and sets up Claude Code config, OpenCode config,
git identity, shell config, and bat theme. It's idempotent — safe to re-run.

For Claude Code online sandboxes, see [Sandbox setup](#sandbox-setup).

### Sandbox setup

In the Claude Environments UI, set:

- **Setup command**: `git clone https://github.com/zalimeni/dotfiles.git ~/dotfiles && ~/dotfiles/install.sh`
- **Environment variables**: `CONTEXT7_KEY`, plus any secrets from `system/.private_env`

## Tools

- **Shell**: zsh + oh-my-zsh + starship
- **Terminal**: Ghostty + tmux
- **Editor**: neovim
- **Go**: go, golangci-lint, staticcheck
- **Utilities**: bat, eza, fzf, ranger, ripgrep, fd, delta
- **Infra**: k9s, mise, gh

## Credits

The [dotfiles community](https://dotfiles.github.io)
