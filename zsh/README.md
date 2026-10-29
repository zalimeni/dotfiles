# Zsh config

This directory holds the zsh4humans (z4h) setup. `install.sh` links each file to `~`.

| File | Linked to | Purpose |
| --- | --- | --- |
| `.zshenv` | `~/.zshenv` | z4h bootstrap. Do not edit. |
| `.zprofile` | `~/.zprofile` | Login-shell environment: Homebrew, PATH, `PNPM_HOME`. |
| `.zshrc` | `~/.zshrc` | z4h template with a small set of changes. |
| `.zshrc_custom` | `~/.zshrc_custom` | All personal customization. |
| `.p10k.zsh` | `~/.p10k.zsh` | Powerlevel10k prompt. |

## Where to put a change

Put personal changes in `.zshrc_custom`. `.zshrc` stays close to the z4h template,
so you can compare it with upstream when z4h changes.

`.zshrc` has one custom line at the end. It sources `~/.zshrc_custom`. That
file runs after `z4h init`, so z4h functions, aliases and `setopt` are available.

These changes stay in `.zshrc`, because they must run before `z4h init`:

- `zstyle ':z4h:bindkey' keyboard 'mac'`
- `zstyle ':z4h:' start-tmux no`
- `zstyle ':z4h:' prompt-at-bottom 'no'`
- The `z4h` availability guards.

Put environment variables and PATH entries for login shells in `.zprofile`.

## Compare with upstream

```sh
curl -fsSL https://raw.githubusercontent.com/romkatv/zsh4humans/v5/.zshrc | diff - zsh/.zshrc
```
