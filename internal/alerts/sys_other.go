//go:build !unix

package alerts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// checkPrivate does nothing where files have no Unix modes. Hosts are Linux;
// this file only lets the package build everywhere.
func checkPrivate(string) error { return nil }

// ownGroup gives a command a short time to finish its output after a timeout.
func ownGroup(cmd *exec.Cmd) { cmd.WaitDelay = 2 * time.Second }

// shell is the program that runs a command destination.
const shell = "sh"

// withLock runs fn with path open. There is no lock: only Linux hosts run
// holders.
func withLock(_ context.Context, path string, fn func(f *os.File) error) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return fn(f)
}
