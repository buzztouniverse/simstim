package host

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/paths"
)

// SystemdUser reports whether a systemd user manager is running for this user.
func SystemdUser() bool {
	out, err := exec.Command("systemctl", "--user", "is-system-running").Output()
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

// launchHolder starts the holder for m and waits until its socket answers.
func launchHolder(m *Manifest) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if SystemdUser() {
		unit := UnitName(m.ID)
		args := []string{"--user", "--quiet", "--collect",
			"--unit=" + unit, "--slice=" + brand.Slice,
			"-p", "KillMode=control-group",
			"-p", "OOMPolicy=continue",
		}
		if m.Limits.MemoryMax > 0 {
			args = append(args,
				"-p", fmt.Sprintf("MemoryMax=%d", m.Limits.MemoryMax),
				"-p", "MemorySwapMax=0")
		}
		args = append(args, exe, "holder", m.ID)
		out, err := exec.Command("systemd-run", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemd-run: %w: %s", err, strings.TrimSpace(string(out)))
		}
		m.Run.Unit = unit
	} else {
		// No systemd user manager: start the holder in its own session, detached.
		devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if err != nil {
			return err
		}
		defer devnull.Close()
		cmd := exec.Command(exe, "holder", m.ID)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			return err
		}
		_ = cmd.Process.Release()
	}
	return waitSocket(m.ID, 5*time.Second)
}

func waitSocket(id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("unix", paths.Socket(id), 200*time.Millisecond); err == nil {
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
		return fmt.Errorf("the session did not start: %s", lines[len(lines)-1])
	}
	return errors.New("the session did not start in time")
}
