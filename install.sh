#!/bin/sh
# sealpup installer — installs the binary and wires it into your shell.
#
#   curl -fsSL https://raw.githubusercontent.com/Photon48/sealpup/main/install.sh | sh
#
# Idempotent and safe to re-run. Respects $SEALPUP_NO_SETUP=1 to skip the shell
# wiring (binary only).
set -eu

MODULE="github.com/Photon48/sealpup"

info() { printf '  %s\n' "$1"; }
err()  { printf 'error: %s\n' "$1" >&2; exit 1; }

printf '\n🦭 Installing sealpup...\n\n'

# 1. Need Go to build. (Prebuilt release binaries are a later addition.)
if ! command -v go >/dev/null 2>&1; then
  err "Go is required to install sealpup.
       Install it from https://go.dev/dl/ and re-run this script."
fi

# 2. Install the binary.
if [ -f "./go.mod" ] && grep -q "$MODULE" "./go.mod" 2>/dev/null; then
  info "building from local checkout"
  go install .
else
  info "downloading and building $MODULE@latest"
  go install "${MODULE}@latest"
fi

# 3. Locate the installed binary.
BINDIR="$(go env GOBIN)"
[ -n "$BINDIR" ] || BINDIR="$(go env GOPATH)/bin"
SEALPUP="$BINDIR/sealpup"
[ -x "$SEALPUP" ] || err "sealpup was not found at $SEALPUP after install"
info "installed to $SEALPUP"

# 4. Wire it into the shell (auto-detects from $SHELL). Skippable.
if [ "${SEALPUP_NO_SETUP:-0}" = "1" ]; then
  printf '\n✓ Binary installed. Shell setup skipped (SEALPUP_NO_SETUP=1).\n'
  printf '  Add this to your shell rc yourself:  eval "$(sealpup init zsh)"\n\n'
  exit 0
fi

printf '\n'
"$SEALPUP" setup

printf '\n✓ Done. Open a new terminal (or run the source command above) and try:\n'
printf '    sealpup new my-first-worktree\n\n'
