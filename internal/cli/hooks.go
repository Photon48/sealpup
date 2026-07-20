package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Photon48/sealpup/internal/config"
	"github.com/Photon48/sealpup/internal/ui"
)

// runCreateHooks applies a repo's .sealpup.toml to a freshly created worktree:
// copy the listed files in from the main worktree, then run post_create inside
// it. cfg is loaded before creation (so a malformed file aborts earlier); this
// runs after. A post_create failure is surfaced but the worktree is kept.
func (e Env) runCreateHooks(cfg *config.Config, mainDir, branch, dir string) error {
	if cfg.IsEmpty() {
		return nil
	}
	e.copyFiles(cfg.Copy, mainDir, dir)
	if cfg.PostCreate != "" {
		return e.runPostCreate(cfg.PostCreate, branch, dir)
	}
	return nil
}

// copyFiles copies each relative path from the main worktree into the new one,
// preserving the relative location and file mode. Best-effort: a missing source
// (a gitignored local file that simply doesn't exist here) is noted and skipped,
// never fatal.
func (e Env) copyFiles(paths []string, mainDir, dir string) {
	p := e.prompter()
	for _, rel := range paths {
		src := filepath.Join(mainDir, rel)
		info, err := os.Stat(src)
		if err != nil {
			p.Infof("note: skipped %s (not in the main worktree)", rel)
			continue
		}
		if info.IsDir() {
			p.Infof("note: skipped %s (copying directories isn't supported yet)", rel)
			continue
		}
		if err := copyFile(src, filepath.Join(dir, rel), info); err != nil {
			p.Infof("note: could not copy %s: %v", rel, err)
			continue
		}
		p.Infof("copied %s", rel)
	}
}

func copyFile(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// runPostCreate runs the post_create command via `sh -c` inside the new
// worktree. Both stdout and stderr of the hook go to stderr — sealpup's stdout
// is reserved for the worktree path (the shim contract). The command line is
// printed first: it's the user's visibility into what code the repo is running.
func (e Env) runPostCreate(command, branch, dir string) error {
	fmt.Fprintf(e.Stderr, "running post_create hook: %s\n", command)

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = e.Stderr
	cmd.Stderr = e.Stderr
	cmd.Stdin = nil

	if err := cmd.Run(); err != nil {
		return ui.Hintf(
			fmt.Sprintf("post_create hook failed (%s); the worktree was still created at %s", exitDesc(err), prettyPath(dir)),
			"fix the hook in .sealpup.toml and rerun it, or just 'sealpup enter "+branch+"' — the worktree already exists",
		)
	}
	return nil
}

// exitDesc renders a hook process failure, preferring the exit code.
func exitDesc(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return fmt.Sprintf("exit %d", ee.ExitCode())
	}
	return err.Error()
}
