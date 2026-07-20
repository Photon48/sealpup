# sealpup 🦭

A light, idiot-proof git worktree manager. Four verbs — `new`, `enter`, `list`,
`delete` — that let you run several branches (and several coding agents) side by
side without them stepping on each other.

---

## ⚡ Quickstart

### Install (one line)

```sh
curl -fsSL https://raw.githubusercontent.com/Photon48/sealpup/main/install.sh | sh
```

That installs the binary **and** wires it into your shell (PATH + the auto-`cd`
shim). Open a new terminal and you're done.

<details>
<summary>Prefer to do it yourself?</summary>

```sh
go install github.com/Photon48/sealpup@latest   # build the binary
sealpup setup                                   # wire it into your shell
```

`sealpup setup` auto-detects your shell (zsh/bash/fish), adds a single managed
block to your rc file, and is safe to re-run. To only print the shim without
touching any files, use `sealpup init zsh`.
</details>

### Try it

From inside any git repo:

```sh
# Spin up a new branch in its own worktree — and you're dropped straight in.
$ sealpup new payments-fix
✓ created worktree for payments-fix at ~/code/app-worktrees/payments-fix
$ pwd
~/code/app-worktrees/payments-fix        # ← you're here now

# Start a coding agent, edit files… then make a second one for parallel work.
$ sealpup new ui-tweak
$ pwd
~/code/app-worktrees/ui-tweak

# See everything at a glance — dirty state and where you are.
$ sealpup list
BRANCH        PATH                                STATUS
main          ~/code/app                          clean
payments-fix  ~/code/app-worktrees/payments-fix   3 files
ui-tweak      ~/code/app-worktrees/ui-tweak       clean    ← you are here

# Hop between them instantly.
$ sealpup enter payments-fix
$ pwd
~/code/app-worktrees/payments-fix

# Done with one? Clean it up (sealpup refuses if you'd lose work).
$ sealpup enter main
$ sealpup delete ui-tweak
✓ removed worktree for ui-tweak
Also delete branch "ui-tweak"? [y/N] y
✓ deleted branch ui-tweak
```

That's the whole tool. 🎉

---

## Why it exists

Git worktrees are great for parallel work but the raw UX fights you: you `cd`
into paths you have to remember, git refuses to check out a branch that's already
live in another worktree with a cryptic error, and deleted worktrees leave
dangling state behind. sealpup wraps the sharp edges so worktrees just click —
and it's built for the 2026 use case of running multiple coding agents at once,
each isolated in its own branch.

## Commands

| Command | What it does |
|---------|--------------|
| `sealpup new <branch>` | Create a branch + worktree under `../<repo>-worktrees/<branch>`, then enter it. |
| `sealpup enter <branch>` | Jump into an existing worktree. |
| `sealpup list` | Show every worktree, its dirty status, and where you are. |
| `sealpup delete <branch>` | Remove a worktree (and optionally its branch). |
| `sealpup init <shell>` | Print the shell integration (`zsh`/`bash`/`fish`). |

`delete` flags: `--force` (remove even if dirty), `--branch` (also delete the
branch, no prompt), `--keep-branch` (delete only the worktree).

## The guardrails (the actual point)

sealpup catches the footguns and *offers the fix* instead of just erroring:

- **`new` a branch that's already checked out** → offers to `enter` it instead.
- **`new` a branch that already exists** → offers to make a worktree for it.
- **`enter` a worktree you `rm -rf`'d** → offers to prune and recreate it.
- **`enter` a branch with no worktree yet** → offers to create one.
- **`delete` a dirty worktree** → refused unless `--force`, and it names the files.
- **`delete` the worktree you're standing in, or `main`** → refused, with a way out.
- Every command prunes dangling worktrees opportunistically, so state self-heals.

## Where worktrees live

Next to your repo, in a sibling directory:

```
~/code/app                    ← main repo
~/code/app-worktrees/
  ├── payments-fix
  └── ui-tweak
```

## How the auto-`cd` works

`new`/`enter` print the target path to **stdout** and everything else (prompts,
progress, errors) to **stderr**. The shell shim captures stdout and runs the
`cd`. Because only stdout is captured, prompts still reach your terminal and
`sealpup list | grep foo` still works. Run `new`/`enter` without the shim and
sealpup still prints the path — it just reminds you how to enable auto-`cd`.

## Not yet (v0.2+)

Detecting which worktrees have coding agents running, per-worktree setup hooks
(`.sealpup.toml` to copy `.env` / run `pnpm install`), shell completions, and
Homebrew packaging.

## Development

```sh
go build ./...
go test ./...     # spins up real, isolated temp git repos
```
