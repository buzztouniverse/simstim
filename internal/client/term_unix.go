//go:build !windows

package client

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// termSize returns the terminal's size in cells and pixels.
func termSize(f *os.File) (cols, rows, xpx, ypx int) {
	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 {
		return 80, 24, 0, 0
	}
	return int(ws.Col), int(ws.Row), int(ws.Xpixel), int(ws.Ypixel)
}

// watchResize delivers a value on the returned channel when the terminal is resized.
func watchResize(_ *os.File) (<-chan struct{}, func()) {
	sig := make(chan os.Signal, 1)
	out := make(chan struct{}, 1)
	signal.Notify(sig, syscall.SIGWINCH)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-sig:
				select {
				case out <- struct{}{}:
				default:
				}
			case <-done:
				return
			}
		}
	}()
	return out, func() { signal.Stop(sig); close(done) }
}

// controlMaster reports whether this OpenSSH can share connections.
const controlMaster = true
