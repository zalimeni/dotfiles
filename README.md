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
  ghostty/        Ghostty terminal config
  k9s/            Kubernetes TUI config
  mise/           mise runtime/tool version manager
  ranger/         ranger file manager
  tmux/           tmux config
```

## Setup

Clone the repo:

```
git clone git@github.com:zalimeni/dotfiles.git ~/workspace/dotfiles
```

Symlink configs as needed:

```sh
# Claude Code
ln -sf ~/workspace/dotfiles/.config/claude/CLAUDE.md ~/.claude/CLAUDE.md
ln -sf ~/workspace/dotfiles/.config/claude/settings.json ~/.claude/settings.json

# bat
ln -sf ~/workspace/dotfiles/bat ~/.config/bat
bat cache --build

# git
# Edit ~/.gitconfig to source from this repo, or symlink directly
```

## Tools

- **Shell**: zsh + oh-my-zsh + starship
- **Terminal**: Ghostty + tmux
- **Editor**: neovim
- **Go**: go, golangci-lint, staticcheck
- **Utilities**: bat, eza, fzf, ranger, ripgrep, fd, delta
- **Infra**: k9s, mise, gh

## Credits

The [dotfiles community](https://dotfiles.github.io)
