package lima

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/iamlakshya/simstim/internal/client"
	"github.com/iamlakshya/simstim/internal/provider"
)

// testdata/list.ndjson is `limactl list --json` from Lima 2.2.0 on macOS
// (one running instance, recorded; one stopped, edited from it), with the
// home folder renamed. The fields are defined in pkg/store/instance.go of
// lima-vm/lima.
func TestParseList(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "list.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	in, err := parseList(b, provider.MachineName("ltest"))
	if err != nil || in == nil {
		t.Fatalf("parse: %v, %v", in, err)
	}
	if in.Status != "Running" || in.SSHConfigFile != "/Users/me/.lima/"+provider.MachineName("ltest")+"/ssh.config" || in.CPUs != 2 || in.Memory != 4<<30 || in.Disk != 40<<30 {
		t.Fatalf("instance = %+v", in)
	}
	other, _ := parseList(b, "other")
	if other == nil || state(other.Status) != client.MachineStopped {
		t.Fatalf("other = %+v", other)
	}
	if none, err := parseList(b, "nope"); none != nil || err != nil {
		t.Fatalf("a missing instance: %v, %v", none, err)
	}
	if none, err := parseList(nil, "nope"); none != nil || err != nil {
		t.Fatalf("no instances: %v, %v", none, err)
	}
}

func TestState(t *testing.T) {
	for in, want := range map[string]string{
		"Running": client.MachineRunning, "Stopped": client.MachineStopped, "Broken": client.MachineStopped,
		"Installing": client.MachineStarting, "": client.MachineStarting,
	} {
		if got := state(in); got != want {
			t.Errorf("state(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestCreateArgs(t *testing.T) {
	got := strings.Join(CreateArgs("vm", provider.Request{}), " ")
	want := "create --tty=false --name=vm --plain --cpus=2 --memory=4 --disk=40 template:ubuntu-lts"
	if got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
	got = strings.Join(CreateArgs("vm", provider.Request{CPUs: 4, MemoryGiB: 8, DiskGiB: 80}), " ")
	if !strings.Contains(got, "--cpus=4 --memory=8 --disk=80") {
		t.Fatal(got)
	}
}

// fakeLimactl keeps each instance's status in a file and logs every call.
const fakeLimactl = `#!/bin/sh
echo "$*" >> "$FAKE_DIR/log"
cmd=$1; shift
case $cmd in
list)
  for f in "$FAKE_DIR"/inst-*; do
    [ -f "$f" ] || continue
    n=${f##*/inst-}
    printf '{"name":"%s","status":"%s","sshConfigFile":"%s/%s/ssh.config","cpus":2}\n' "$n" "$(cat "$f")" "$FAKE_DIR" "$n"
  done ;;
create)
  for a; do case $a in --name=*) n=${a#--name=} ;; esac; done
  echo Stopped > "$FAKE_DIR/inst-$n" ;;
start) for a; do n=$a; done; echo Running > "$FAKE_DIR/inst-$n" ;;
stop) echo Stopped > "$FAKE_DIR/inst-$1" ;;
delete) for a; do n=$a; done; rm -f "$FAKE_DIR/inst-$n" ;;
*) echo "fake limactl: unknown command $cmd" >&2; exit 2 ;;
esac
`

func TestCreateAndLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake limactl is a POSIX shell script")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "limactl"), []byte(fakeLimactl), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_DIR", dir)
	ctx := context.Background()

	h, err := Provider{}.Create(ctx, provider.Request{Name: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	inst := provider.MachineName("dev")
	if h.SSH != "lima-"+inst || h.SSHConfig != filepath.Join(dir, inst, "ssh.config") || h.Machine.Provider != "lima" || h.Machine.ID != inst || h.Machine.IdleStop != DefaultIdleStop {
		t.Fatalf("host = %+v, machine = %+v", h, h.Machine)
	}
	log, _ := os.ReadFile(filepath.Join(dir, "log"))
	for _, want := range []string{"create --tty=false --name=" + inst + " --plain --cpus=2 --memory=4 --disk=40 template:ubuntu-lts", "start --tty=false " + inst} {
		if !strings.Contains(string(log), want) {
			t.Fatalf("limactl calls:\n%s\nwant %q", log, want)
		}
	}
	if _, err := (Provider{}).Create(ctx, provider.Request{Name: "dev"}); err == nil {
		t.Fatal("a second instance of the same name was made")
	}

	lc, _ := Provider{}.Open(h)
	status := func() string {
		s, err := lc.Status(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	if status() != client.MachineRunning {
		t.Fatal(status())
	}
	tr, err := lc.Transport(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := tr.(*client.SSH)
	if !ok || s.Host.SSH != "lima-"+inst || s.Host.SSHConfig != h.SSHConfig || !s.Interactive {
		t.Fatalf("transport = %#v", tr)
	}
	if err := lc.Stop(ctx); err != nil || status() != client.MachineStopped {
		t.Fatalf("stop: %v, %s", err, status())
	}
	if err := lc.Start(ctx); err != nil || status() != client.MachineRunning {
		t.Fatalf("start: %v, %s", err, status())
	}
	if err := lc.Destroy(ctx); err != nil || status() != client.MachineMissing {
		t.Fatalf("destroy: %v, %s", err, status())
	}
	if err := lc.Destroy(ctx); err != nil {
		t.Fatalf("destroying a missing VM: %v", err)
	}
}
