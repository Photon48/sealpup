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

On **bash** the block lands in `~/.bashrc`, and setup also makes your login file
(`~/.bash_profile`, else `~/.bash_login`/`~/.profile`) source `~/.bashrc`. That's
what makes the pre-installed bash on macOS work: a Terminal session there is a
*login* shell, which reads `~/.bash_profile` and never `~/.bashrc` on its own.
Setup leaves your login file untouched if it already sources `~/.bashrc`.
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

# See everything at a glance — dirty state, where you are, and which
# worktrees have a coding agent running in them.
$ sealpup list
BRANCH        PATH                               STATUS   AGENT
main          ~/code/app                         clean    -
payments-fix  ~/code/app-worktrees/payments-fix  3 files  claude (pid 4812)
ui-tweak      ~/code/app-worktrees/ui-tweak      clean    -                  ← you are here

# Hop between them instantly.
$ sealpup enter payments-fix
$ pwd
~/code/app-worktrees/payments-fix

# Done with one? Clean it up (sealpup refuses if you'd lose work — or if
# an agent is still running there).
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
| `sealpup update` | Upgrade sealpup to the latest release. |

`delete` flags: `--force` (remove even if dirty), `--branch` (also delete the
branch, no prompt), `--keep-branch` (delete only the worktree).

## The guardrails (the actual point)

sealpup catches the footguns and *offers the fix* instead of just erroring:

- **`new` a branch that's already checked out** → offers to `enter` it instead.
- **`new` a branch that already exists** → offers to make a worktree for it.
- **`enter` a worktree you `rm -rf`'d** → offers to prune and recreate it.
- **`enter` a branch with no worktree yet** → offers to create one.
- **`delete` a worktree with an agent running in it** → refused unless `--force`, naming the agent + pid.
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

## Agent detection

`sealpup list` shows which worktrees have a coding agent running, and `delete`
refuses to pull a worktree out from under one. Detection is live — sealpup scans
running processes and matches those whose working directory is inside a worktree.
There's **no state to track and nothing to go stale**: it works even for agents
you started without sealpup, and clears the moment they exit.

Recognized out of the box: `claude`, `aider`, `codex`, `cursor-agent`, `gemini`,
`amp`, `goose`, `opencode`, `copilot`. Add your own:

```sh
export SEALPUP_AGENTS="mytool,another-agent"   # merged with the built-in list
```

Works on macOS and Linux. (On other platforms `list` simply shows no agents.)

## Per-worktree setup with `.sealpup.toml`

A fresh worktree has no `node_modules`, no `.env` — useless to an agent until you
set it up. Drop a `.sealpup.toml` at your repo root and sealpup does it on every
`new`:

```toml
[hooks]
# Files copied from the main worktree into each new one (great for gitignored
# local config that isn't committed).
copy = [".env", ".env.local"]

# A command run inside the new worktree right after it's created.
post_create = "pnpm install"
```

That's the whole format — only `[hooks]` with `copy` (array of strings) and
`post_create` (string). Anything else is rejected with the offending line number,
so a bad file fails loudly instead of being silently ignored. Missing `copy`
sources are skipped with a note (local files legitimately may not exist yet). If
`post_create` fails, the worktree is kept — fix the hook and `sealpup enter` it.

> **Trust note:** `post_create` runs a command from a file committed to the repo.
> It's the same trust model as `npm install` postinstall scripts or a Makefile —
> only run `sealpup new` in repos whose code you'd already run. sealpup always
> prints the command before executing it.

## Staying up to date

sealpup checks for new releases at most **once an hour**, in the background, and
never blocks a command on the network: it asks the Go module proxy (the same
source `go install @latest` resolves) and caches the answer. When a newer
release exists you get a one-line reminder on stderr:

```
🦭 sealpup v0.3.0 is available (you have v0.2.0) — run `sealpup update`
```

`sealpup update` reinstalls the latest release the same way the installer does
(re-running the curl one-liner works too). To disable the check entirely:

```sh
export SEALPUP_NO_UPDATE_CHECK=1
```

## How the auto-`cd` works

`new`/`enter` print the target path to **stdout** and everything else (prompts,
progress, errors, hook output) to **stderr**. The shell shim captures stdout and
runs the `cd`. Because only stdout is captured, prompts still reach your terminal
and `sealpup list | grep foo` still works. Run `new`/`enter` without the shim and
sealpup still prints the path — it just reminds you how to enable auto-`cd`.

## Not yet (v0.3+)

Globs and directory copies in `.sealpup.toml`, a direnv-style trust prompt for
`post_create`, showing the full agent list per worktree, Windows support, shell
completions, and Homebrew packaging.

## Development

```sh
go build ./...
go test ./...     # spins up real, isolated temp git repos
```
