# Login-shell environment — linked to ~/.zprofile by install.sh.
# Interactive-only setup (aliases, prompt, tool hooks) lives in .zshrc.

# Homebrew: sets PATH, MANPATH, HOMEBREW_* variables.
if [[ -x /opt/homebrew/bin/brew ]]; then
  eval "$(/opt/homebrew/bin/brew shellenv zsh)"
fi

# Keep each PATH entry once, even if this file runs again.
typeset -U path

# User-installed tools.
path=(~/.local/bin $path)

# pnpm
export PNPM_HOME="$HOME/Library/pnpm"
path=($PNPM_HOME $path)
