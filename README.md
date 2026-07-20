# sealpup

A light, idiot-proof git worktree manager. Four verbs — `new`, `enter`, `list`,
`delete` — and error messages that offer the fix instead of just naming the
problem. Built for running several coding agents in parallel, each in its own
worktree, without them stepping on each other.

```
$ sealpup new payments-fix       # create branch + worktree, and drop you in it
✓ created worktree for payments-fix at ~/code/app-worktrees/payments-fix

$ sealpup list
BRANCH        PATH                              STATUS
main          ~/code/app                        clean
payments-fix  ~/code/app-worktrees/payments-fix 3 files  ← you are here
ui-refactor   ~/code/app-worktrees/ui-refactor  clean

$ sealpup enter ui-refactor      # just cd's you there
$ sealpup delete payments-fix    # removes the worktree (refuses if dirty)
```

## Why

Git worktrees are great for parallel work but the raw UX fights you: you `cd`
into paths you have to remember, git refuses to check out a branch that's live
in another worktree with a cryptic error, and deleted worktrees leave dangling
state behind. sealpup wraps the sharp edges so worktrees just click.

## Install

Requires `git` and Go 1.24+ (to build).

```sh
go install github.com/Photon48/sealpup@latest
```

Then add the shell integration so `new`/`enter` can change your directory (a
binary can't `cd` its parent shell on its own — this one-line shim does it, the
same trick zoxide and direnv use):

```sh
# zsh — add to ~/.zshrc
eval "$(sealpup init zsh)"

# bash — add to ~/.bashrc
eval "$(sealpup init bash)"

# fish — add to ~/.config/fish/config.fish
sealpup init fish | source
```

Reload your shell and you're set.

## Commands

| Command | What it does |
|---------|--------------|
| `sealpup new <branch>` | Create a branch and its worktree under `../<repo>-worktrees/<branch>`, then enter it. |
| `sealpup enter <branch>` | Jump into an existing worktree. |
| `sealpup list` | Show every worktree, its dirty status, and where you are. |
| `sealpup delete <branch>` | Remove a worktree (and optionally its branch). |
| `sealpup init <shell>` | Print the shell integration (`zsh`/`bash`/`fish`). |

`delete` flags: `--force` (remove even if dirty), `--branch` (also delete the
branch, no prompt), `--keep-branch` (delete only the worktree).

## The guardrails

This is the actual product — sealpup catches the footguns and offers the fix:

- **`new` a branch that's already checked out** → offers to `enter` it instead.
- **`new` a branch that already exists** → offers to make a worktree for it.
- **`enter` a worktree whose directory was `rm -rf`'d** → offers to prune and recreate it.
- **`enter` a branch with no worktree yet** → offers to create one.
- **`delete` a dirty worktree** → refused unless `--force`, and it tells you which files.
- **`delete` the worktree you're standing in, or the main checkout** → refused, with a way out.
- Every command runs `git worktree prune` opportunistically, so dangling state self-heals.

## How the `cd` works

`sealpup new`/`enter` print the target worktree path to **stdout** and everything
else (prompts, progress, errors) to **stderr**. The shell shim captures stdout
and runs the `cd` for you. Because only stdout is captured, prompts still reach
your terminal and `sealpup list | grep foo` still works. If you run `new`/`enter`
without the shim installed, sealpup still prints the path and reminds you how to
enable auto-`cd`.

## Layout

Worktrees live in a sibling directory next to your repo:

```
~/code/app                    ← main repo
~/code/app-worktrees/
  ├── payments-fix
  └── ui-refactor
```

## Not yet (v0.2+)

Detecting which worktrees have coding agents running, per-worktree setup hooks
(`.sealpup.toml` to copy `.env` / run `pnpm install`), shell completions, and
Homebrew packaging.

## Development

```sh
go build ./...
go test ./...
```

Tests spin up real, fully isolated temp git repos (`internal/git/gittest`), so
they exercise the actual git you'd run against.
