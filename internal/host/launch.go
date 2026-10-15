package host

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/iamlakshya/simstim/internal/brand"
	"github.com/iamlakshya/simstim/internal/paths"
)

// SystemdUser reports whether a systemd user manager is running for this user.
func SystemdUser() bool {
	out, err := exec.CommandContext(context.Background(), "systemctl", "--user", "is-system-running").Output()
	s := strings.TrimSpace(string(out))
	if err != nil && s == "" {
		return false
	}
	return s == "running" || s == "degraded" || s == "starting"
}

// MemTotal returns the host's memory in bytes.
func MemTotal() int64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "MemTotal:"); ok {
			v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "kB"))
			n, _ := strconv.ParseInt(v, 10, 64)
			return n * 1024
		}
	}
	return 0
}

// DefaultMemoryMax is 8 GiB or 40 % of the host's memory, whichever is smaller.
func DefaultMemoryMax() int64 {
	const cap8 = 8 << 30
	total := MemTotal()
	if total <= 0 {
		return cap8
	}
	if v := total * 40 / 100; v < cap8 {
		return v
	}
	return cap8
}

// UnitName is the systemd unit of a session.
func UnitName(id string) string { return brand.UnitPrefix + id + ".service" }

// holderExecutable is the helper binary that holders are started from.
// Tests point it at a copy of the test binary with a shadow terminal next
// to it.
var holderExecutable = os.Executable

// launchHolder starts the holder for m and waits until its socket answers.
func launchHolder(m *Manifest) error {
	exe, err := holderExecutable()
	if err != nil {
		return err
	}
	if SystemdUser() {
		unit := UnitName(m.ID)
		args := []string{"--user", "--quiet", "--collect",
			"--unit=" + unit, "--slice=" + brand.Slice,
			// SIGTERM goes to the holder only, which stops its agent itself and
			// records why; anything left is killed after the timeout.
			"-p", "KillMode=mixed",
			"-p", "TimeoutStopSec=15",
			"-p", "OOMPolicy=continue",
		}
		if m.Limits.MemoryMax > 0 {
			args = append(args,
				"-p", fmt.Sprintf("MemoryMax=%d", m.Limits.MemoryMax),
				"-p", "MemorySwapMax=0")
		}
		args = append(args, exe, "holder", m.ID)
		out, err := exec.CommandContext(context.Background(), "systemd-run", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemd-run: %w: %s", err, strings.TrimSpace(string(out)))
		}
		m.Run.Unit = unit
	} else {
		// No systemd user manager: start the holder in its own session, detached.
		devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("start the holder: %w", err)
		}
		defer devnull.Close()
		cmd := exec.CommandContext(context.Background(), exe, "holder", m.ID)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start the holder: %w", err)
		}
		_ = cmd.Process.Release()
	}
	return waitSocket(m.ID, 5*time.Second)
}

func waitSocket(id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		d := net.Dialer{Timeout: 200 * time.Millisecond}
		if c, err := d.DialContext(context.Background(), "unix", paths.Socket(id)); err == nil {
			c.Close()
			return nil
		}
		// The holder writes an errored status if the agent failed to start.
		if m, err := LoadManifest(id); err == nil && m.Status.State == StateErrored {
			return errors.New(m.Status.Reason)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if b, err := os.ReadFile(paths.HolderLog(id)); err == nil && len(b) > 0 {
		lines := strings.Split(strings.TrimSpace(string(b)), "\n")
		return fmt.Errorf("the session did not start: %s", logText(lines[len(lines)-1]))
	}
	return errors.New("the session did not start in time")
}

// logText turns a line of a holder's log into text for the user: its message
// and its error, without the time and the level. A line in another form, such
// as one an older helper wrote, comes back as it is.
func logText(line string) string {
	var msg, errText string
	for rest := line; rest != ""; {
		key, val, ok := strings.Cut(rest, "=")
		if !ok {
			break
		}
		if strings.HasPrefix(val, `"`) {
			q, err := strconv.QuotedPrefix(val)
			if err != nil {
				break
			}
			rest = strings.TrimPrefix(val[len(q):], " ")
			val, _ = strconv.Unquote(q)
		} else {
			val, rest, _ = strings.Cut(val, " ")
		}
		switch key {
		case "msg":
			msg = val
		case "err":
			errText = val
		}
	}
	switch {
	case msg == "":
		return line
	case errText == "":
		return msg
	}
	return msg + ": " + errText
}
