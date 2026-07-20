// Command sealpup is a light, idiot-proof git worktree manager.
package main

import (
	"os"

	"github.com/Photon48/sealpup/internal/cli"
)

func main() {
	os.Exit(cli.Run(cli.RealEnv(), os.Args[1:]))
}
