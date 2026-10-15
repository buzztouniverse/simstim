package host

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/iamlakshya/simstim/internal/brand"
)

// fakeProc points procRoot at a folder of files, for the rest of the test.
// Each key is a path under it; a value that starts with "->" is a link.
func fakeProc(t *testing.T, files map[string]string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "proc")
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		var err error
		if target, ok := strings.CutPrefix(body, "->"); ok {
			err = os.Symlink(target, p)
		} else {
			err = os.WriteFile(p, []byte(body), 0o600)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	old := procRoot
	procRoot = root
	t.Cleanup(func() { procRoot = old })
}

const tcpHeader = "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n"

func tcpLine(local, state, inode string) string {
	return "   0: " + local + " 00000000:0000 " + state + " 00000000:00000000 00:00000000 00000000  1000        0 " + inode + " 1 0000000000000000 100 0 0 10 0\n"
}

func TestSessionPorts(t *testing.T) {
	fakeProc(t, map[string]string{
		// The agent (100), its server (101) and the server's worker (102),
		// and someone else's process (200).
		"100/stat":    "100 (sh) S 1 100 100 0",
		"101/stat":    "101 (node server) S 100 100 100 0",
		"102/stat":    "102 (postgres) S 101 100 100 0",
		"200/stat":    "200 (other) S 1 200 200 0",
		"self/stat":   "garbage",
		"101/comm":    "node\n",
		"102/comm":    "postgres\n",
		"101/fd/3":    "->socket:[5555]",
		"101/fd/4":    "->socket:[6666]",
		"101/fd/5":    "->/dev/null",
		"102/fd/3":    "->socket:[8888]",
		"200/fd/3":    "->socket:[7777]",
		"self/cgroup": "0::/user.slice/no-such.service\n",
		"net/tcp": tcpHeader +
			tcpLine("0100007F:0BB8", "0A", "5555") + // 127.0.0.1:3000, listening
			tcpLine("0100007F:0BB9", "01", "6666") + // a connection, not a listener
			tcpLine("00000000:1F90", "0A", "7777") + // :8080, not the session's
			"short line\n",
		"net/tcp6": tcpHeader +
			tcpLine("00000000000000000000000001000000:1538", "0A", "8888") + // [::1]:5432
			tcpLine("00000000000000000000000000000000:0BB8", "0A", "5555"), // 3000 again
	})
	pids := sessionPIDs(false, 100)
	if !reflect.DeepEqual(pids, []int{100, 101, 102}) {
		t.Fatalf("the agent and its descendants = %v", pids)
	}
	want := []Port{
		{Port: 3000, Address: "127.0.0.1", PID: 101, Process: "node"},
		{Port: 5432, Address: "::1", PID: 102, Process: "postgres"},
	}
	if got := listeningPorts(pids); !reflect.DeepEqual(got, want) {
		t.Errorf("ports = %+v, want %+v", got, want)
	}
	// A session in a systemd unit is its unit's cgroup; here there is none.
	if got := sessionPIDs(true, 100); len(got) != 0 {
		t.Errorf("pids of a missing cgroup = %v", got)
	}
}

func TestIdleCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_RUNTIME_DIR", home)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	fakeProc(t, map[string]string{})

	for _, args := range [][]string{nil, {"setup"}, {"setup", "10s"}, {"setup", "soon"}, {"setup", "30m"}, {"check", "--after", "soon"}, {"bogus"}} {
		if msg, err := RunIdle(args); err == nil {
			t.Errorf("idle %q = %q, want an error", args, msg)
		}
	}
	for _, args := range [][]string{{"status"}, {"check"}, {"setup", "never"}} {
		if msg, err := RunIdle(args); err != nil || msg != "idle stop is off" {
			t.Errorf("idle %q = %q, %v", args, msg, err)
		}
	}
	if IdleDoctor() != nil {
		t.Error("the doctor reports idle stop while it is off")
	}

	// With the service installed, as `idle setup` writes it.
	service, _ := IdleUnits("/helper", 30*time.Minute)
	if err := os.MkdirAll(userUnitDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userUnitDir(), idleUnit("service")), []byte(service), 0o600); err != nil {
		t.Fatal(err)
	}
	if msg, err := RunIdle([]string{"status"}); err != nil || !strings.HasPrefix(msg, "idle stop after 30m0s; passwordless sudo: ") {
		t.Errorf("status = %q, %v", msg, err)
	}
	if d := IdleDoctor(); len(d) != 1 || d[0].Name != "idle stop" || !strings.Contains(d[0].Detail, "30m0s") {
		t.Errorf("doctor = %+v", d)
	}
	// The first check starts the count. It never powers off here: the host
	// was busy a moment ago.
	if msg, err := RunIdle([]string{"check"}); err != nil || !strings.HasPrefix(msg, "busy 0s ago (clients 0, working false); powers off after 30m0s") {
		t.Errorf("check = %q, %v", msg, err)
	}
	if _, err := os.Stat(idleStatePath()); err != nil {
		t.Errorf("the check kept no state: %v", err)
	}
	if msg, err := RunIdle([]string{"setup", "never"}); err != nil || msg != "idle stop is off" {
		t.Errorf("setup never = %q, %v", msg, err)
	}
	if _, err := os.Stat(filepath.Join(userUnitDir(), idleUnit("service"))); !os.IsNotExist(err) {
		t.Error("setup never left the service")
	}
}

func TestIdleCheckCountsClients(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	fakeProc(t, map[string]string{
		"300/cmdline":  "/home/dev/.local/share/bin/v1/" + brand.HostBinary + "\x00bridge\x00",
		"301/cmdline":  brand.HostBinary + "\x00attach\x00abc\x00",
		"302/cmdline":  brand.HostBinary + "\x00holder\x00abc\x00",
		"303/cmdline":  "sshd\x00",
		"self/cmdline": brand.HostBinary + "\x00bridge\x00",
		"400/stat":     "no cmdline here",
	})
	now := time.Now()
	msg, err := IdleCheck(time.Hour, now)
	if err != nil || !strings.Contains(msg, "clients 2,") {
		t.Fatalf("check = %q, %v", msg, err)
	}
	// An hour of quiet has not passed: still no power off.
	if msg, err = IdleCheck(time.Hour, now.Add(time.Minute)); err != nil || !strings.HasPrefix(msg, "busy 0s ago") {
		t.Fatalf("check a minute later = %q, %v", msg, err)
	}
}
