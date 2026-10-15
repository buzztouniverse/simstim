//go:build unix

package alerts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// checkPrivate fails when a file that may hold a secret can be read by
// anyone but its owner, or belongs to someone else.
func checkPrivate(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return &Error{Msg: "cannot read " + path + ": " + err.Error(), Hint: "Check that the file exists."}
	}
	if st.Mode().Perm()&0o077 != 0 {
		return &Error{Msg: fmt.Sprintf("%s can be read by other users (mode %04o), so it is not used", path, st.Mode().Perm()),
			Hint: "chmod 600 " + path}
	}
	if sys, ok := st.Sys().(*syscall.Stat_t); ok && int(sys.Uid) != os.Getuid() {
		return &Error{Msg: path + " belongs to another user, so it is not used", Hint: "chown $USER " + path}
	}
	return nil
}

// ownGroup runs cmd in a process group of its own, and makes a timeout stop
// the whole group, so a command's children cannot outlive it.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 2 * time.Second
}

// shell is the program that runs a command destination.
const shell = "/bin/sh"

// withLock runs fn with path open and locked against every other holder on
// the host. It gives up when ctx ends rather than hold up a delivery.
func withLock(ctx context.Context, path string, fn func(f *os.File) error) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EINTR) {
			return fmt.Errorf("lock %s: %w", path, err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("lock %s: %w", path, ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
	defer func() { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN) }()
	return fn(f)
}
