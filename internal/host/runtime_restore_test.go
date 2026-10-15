package host_test

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/iamlakshya/simstim/internal/host"
	"github.com/iamlakshya/simstim/internal/host/hosttest"
)

// crash kills a session's holder, as a crash or a power cut would.
func crash(t *testing.T, id string) {
	t.Helper()
	pid := hosttest.HolderPID(t, id)
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	if err := hosttest.Reap(pid, 5*time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestRestore(t *testing.T) {
	hosttest.New(t)
	// A shell session: it continues with a new login shell.
	s, err := host.New(host.NewArgs{Agent: "shell", Name: "sh", Cwd: "~"})
	if err != nil {
		t.Fatal(err)
	}
	// It ends a moment after New returns: New saves the manifest once more
	// after the holder has started, and an end before that save is lost.
	ended := start(t, "ended", "/bin/sh", "-c", "sleep 0.5; exit 0")
	waitState(t, ended.ID, host.StateExited)
	other := start(t, "no-policy", "sleep", "60")
	tired := start(t, "tired", "sleep", "60")
	for _, id := range []string{s.ID, ended.ID, other.ID, tired.ID} {
		crash(t, id)
	}
	m := manifest(t, other.ID)
	m.Restart.Policy = "never"
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	m = manifest(t, tired.ID)
	m.Run.RespawnCount = m.Restart.MaxAttempts
	m.Run.BootID = "a boot before"
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}

	// Without --all, only sessions from an earlier boot come back; this
	// boot's crash needs --all.
	why := map[string]string{}
	for _, r := range host.Restore(false) {
		why[r.ID] = r.Action + ": " + r.Why
	}
	want := map[string]string{
		s.ID:     "skipped: no restart since it stopped",
		ended.ID: "skipped: the agent had ended",
		other.ID: "skipped: restart policy is never",
		tired.ID: "skipped: already restored 3 times",
	}
	for id, w := range want {
		if !strings.HasPrefix(why[id], w) {
			t.Errorf("%s: %q, want %q", id, why[id], w)
		}
	}

	res := host.Restore(true)
	var got *host.RestoreResult
	for i := range res {
		if res[i].ID == s.ID {
			got = &res[i]
		}
	}
	if got == nil || got.Action != "restored" || got.Why != "continued the latest conversation in its folder" || len(got.Argv) != 2 || got.Argv[0] != "/bin/sh" {
		t.Fatalf("restore --all = %+v", res)
	}
	m = manifest(t, s.ID)
	if m.Run.RespawnCount != 1 || m.Labels["restored_at"] == "" || m.Run.HolderPID == 0 {
		t.Errorf("manifest after restore: %+v %+v", m.Run, m.Labels)
	}
	send(t, s.ID, "echo BACK-$((5*5))")
	waitOutput(t, s.ID, "BACK-25")
	// A live session is left alone.
	for _, r := range host.Restore(true) {
		if r.ID == s.ID {
			t.Errorf("a live session was looked at again: %+v", r)
		}
	}
}

func TestRestoreResumesTheAgentsOwnSession(t *testing.T) {
	h := hosttest.New(t)
	fakeClaude(t, h, "exec sleep 60")
	s, err := host.New(host.NewArgs{Agent: "claude", Cwd: "~"})
	if err != nil {
		t.Fatal(err)
	}
	m := manifest(t, s.ID)
	if m.Agent.IDSource != "chosen" || m.Agent.AgentSessionID == "" || m.Agent.Argv[2] != m.Agent.AgentSessionID {
		t.Fatalf("claude gets its session ID chosen for it: %+v", m.Agent)
	}
	crash(t, s.ID)
	m.Run.BootID = "a boot before"
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	res := host.Restore(false)
	if len(res) != 1 || res[0].Action != "restored" || !strings.Contains(res[0].Why, m.Agent.AgentSessionID) {
		t.Fatalf("restore = %+v", res)
	}
	if res[0].Argv[1] != "--resume" || res[0].Argv[2] != m.Agent.AgentSessionID || res[0].Argv[3] != "--settings" {
		t.Errorf("argv = %q: resume its own session, with the hooks again", res[0].Argv)
	}
}

// A restore whose agent cannot start says so, rather than "restored": the
// holder answers on its socket before it starts the agent, so Restore waits
// for the start.
func TestRestoreReportsAnAgentThatCannotStart(t *testing.T) {
	h := hosttest.New(t)
	s := start(t, "gone", "sleep", "60")
	crash(t, s.ID)
	m := manifest(t, s.ID)
	// The program was removed while the host was down.
	m.Agent.Argv = []string{filepath.Join(h.Home, "removed-program")}
	m.Run.BootID = "a boot before"
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	res := host.Restore(false)
	if len(res) != 1 || res[0].Action != "failed" || !strings.Contains(res[0].Why, "no such file") {
		mm := manifest(t, s.ID)
		t.Fatalf("restore = %+v, want failed with the holder's reason; status %+v, run %+v", res, mm.Status, mm.Run)
	}
}

// A restored session starts at once without its secrets, which nobody can
// send after a restart, even when the restore comes within a second of the
// last start.
func TestRestoredHolderDoesNotWaitForSecrets(t *testing.T) {
	hosttest.New(t)
	s := start(t, "with-secret", "sleep", "60")
	crash(t, s.ID)
	m := manifest(t, s.ID)
	m.Secrets = []string{"TEST_TOKEN"}
	m.Run.BootID = "a boot before"
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	if res := host.Restore(false); len(res) != 1 || res[0].Action != "restored" {
		t.Fatalf("restore = %+v", res)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		mm := manifest(t, s.ID)
		if mm.Run.AgentPID != 0 && len(mm.MissingSecrets) == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("after 3 s the agent has not started without its secrets: run %+v, missing %v", mm.Run, mm.MissingSecrets)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
