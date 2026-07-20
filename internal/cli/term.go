package cli

import "os"

// isTerminal reports whether f is a character device (a TTY), used to decide
// whether prompts and color are safe.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
