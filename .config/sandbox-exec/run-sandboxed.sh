#!/usr/bin/env bash
set -euo pipefail

PROFILE_PATH="$HOME/.config/sandbox-exec/agent.sb"
KEEP_DEBUG_PROFILE="${SANDBOX_EXEC_KEEP_PROFILE:-0}"

usage() {
    printf 'Usage: %s [--workdir=/path] [--debug-profile] <command> [args...]\n' "${0##*/}" >&2
}

sb_escape() {
    printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

sed_escape_replacement() {
    printf '%s' "$1" | sed 's/[\\&|]/\\&/g'
}

resolve_workdir() {
    local requested="${1:-}"
    local git_root=""

    if [[ -n "$requested" ]]; then
        if [[ ! -d "$requested" ]]; then
            printf 'Workdir does not exist: %s\n' "$requested" >&2
            exit 1
        fi
        (cd "$requested" && pwd -P)
        return 0
    fi

    if git_root="$(git rev-parse --show-toplevel 2>/dev/null)" && [[ -n "$git_root" ]]; then
        (cd "$git_root" && pwd -P)
        return 0
    fi

    pwd -P
}

emit_ancestor_literals() {
    local path="$1"
    local current=""
    local part
    local trimmed="${path#/}"

    printf ';; Generated ancestor directory literals for selected workdir: %s\n' "$path"
    printf ';; Wrapper-side injection mirrored from Agent Safehouse emit_path_ancestor_literals().\n'
    printf '(allow file-read*\n'
    printf '    (literal "/")                                                  ;; Ancestor traversal for %s.\n' "$path"

    IFS='/' read -r -a parts <<< "$trimmed"
    for part in "${parts[@]}"; do
        [[ -n "$part" ]] || continue
        current="${current}/$part"
        printf '    (literal "%s") ;; Ancestor traversal for %s.\n' "$(sb_escape "$current")" "$path"
    done

    printf ')\n'
}

workdir=""
debug_profile=0
while [[ $# -gt 0 ]]; do
    case "$1" in
        --debug-profile)
            debug_profile=1
            shift
            ;;
        --workdir=*)
            workdir="${1#*=}"
            shift
            ;;
        --workdir)
            [[ $# -ge 2 ]] || { usage; exit 1; }
            workdir="$2"
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            break
            ;;
    esac
done

[[ $# -gt 0 ]] || { usage; exit 1; }
[[ -f "$PROFILE_PATH" ]] || { printf 'Missing profile: %s\n' "$PROFILE_PATH" >&2; exit 1; }
command -v sandbox-exec >/dev/null 2>&1 || { printf 'sandbox-exec not found in PATH\n' >&2; exit 1; }

resolved_workdir="$(resolve_workdir "$workdir")"
ancestor_file="$(mktemp)"
replaced_profile="$(mktemp)"
before_marker="$(mktemp)"
after_marker="$(mktemp)"
assembled_profile="$(mktemp -t sandbox-exec-profile)"
cleanup() {
    rm -f "$ancestor_file" "$replaced_profile" "$before_marker" "$after_marker"
    if [[ "$debug_profile" -eq 0 && "$KEEP_DEBUG_PROFILE" != "1" ]]; then
        rm -f "$assembled_profile"
    fi
}
trap cleanup EXIT

emit_ancestor_literals "$resolved_workdir" > "$ancestor_file"
escaped_workdir="$(sed_escape_replacement "$resolved_workdir")"
sed "s|__SAFEHOUSE_WORKDIR__|$escaped_workdir|g" "$PROFILE_PATH" > "$replaced_profile"
sed '/;; __WORKDIR_BLOCK_START__/,$d' "$replaced_profile" > "$before_marker"
sed -n '/;; __WORKDIR_BLOCK_START__/,$p' "$replaced_profile" > "$after_marker"
cat "$before_marker" "$ancestor_file" "$after_marker" > "$assembled_profile"

if [[ "$debug_profile" -eq 1 || "$KEEP_DEBUG_PROFILE" = "1" ]]; then
    printf 'Assembled sandbox profile: %s\n' "$assembled_profile" >&2
fi

exec sandbox-exec -p "$(cat "$assembled_profile")" "$@"
