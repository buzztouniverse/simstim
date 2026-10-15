package host_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/host/hosttest"
	"github.com/iAmLakshya/simstim/internal/paths"
	"github.com/iAmLakshya/simstim/internal/proto"
)

func TestSessionLifecycle(t *testing.T) {
	h := hosttest.New(t)
	s := start(t, "first", "/bin/sh", "-c", "echo READY-$((40+2)); exec sleep 60")
	if s.ID == "" || s.Name != "first" || s.Agent != "custom" || !s.Alive || (s.State != host.StateStarting && s.State != host.StateWorking) {
		t.Fatalf("new returned %+v", s)
	}
	if s.Cwd != h.Home {
		t.Errorf("cwd = %q, want the home folder %q", s.Cwd, h.Home)
	}
	waitOutput(t, s.ID, "READY-42")

	list, err := host.List()
	if err != nil || len(list) != 1 || list[0].ID != s.ID || !list[0].Alive {
		t.Fatalf("list = %+v, %v", list, err)
	}
	m := live(t, s.ID)
	if m.Run.HolderPID == 0 || m.Run.AgentPID == 0 || m.Run.Socket != paths.Socket(s.ID) || m.HolderBinary == "" {
		t.Errorf("the holder did not record itself: %+v", m.Run)
	}
	if fi, err := os.Stat(paths.Socket(s.ID)); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("socket: %v, %v", fi, err)
	}

	res, err := host.Delete(s.ID)
	if err != nil || !res.Deleted {
		t.Fatalf("delete = %+v, %v", res, err)
	}
	if err := hosttest.Reap(m.Run.HolderPID, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.SessionDir(s.ID)); !os.IsNotExist(err) {
		t.Fatalf("the session's folder is still there: %v", err)
	}
	if list, _ := host.List(); len(list) != 0 {
		t.Fatalf("list after delete = %+v", list)
	}
}

func TestListIsNewestFirstAndShowsEndedSessions(t *testing.T) {
	hosttest.New(t)
	if list, err := host.List(); err != nil || len(list) != 0 {
		t.Fatalf("list with no sessions = %+v, %v", list, err)
	}
	// It ends a moment after New returns: New saves the manifest once more
	// after the holder has started, and an end before that save is lost.
	a := start(t, "a", "/bin/sh", "-c", "sleep 0.5; exit 3")
	b := start(t, "b", "sleep", "60")
	waitState(t, a.ID, host.StateErrored)
	list, err := host.List()
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if list[0].ID != b.ID || list[1].ID != a.ID {
		t.Errorf("order = %s, %s; want the newest first", list[0].Name, list[1].Name)
	}
	ended := list[1]
	if ended.State != host.StateErrored || ended.ExitCode == nil || *ended.ExitCode != 3 {
		t.Errorf("an agent that exited with 3: %+v", ended)
	}
	if !ended.Alive {
		t.Errorf("the holder of an ended agent stays, so the output can be read: %+v", ended)
	}
}

func TestNewChecksItsArguments(t *testing.T) {
	h := hosttest.New(t)
	for _, c := range []struct {
		args host.NewArgs
		code string
	}{
		{host.NewArgs{Agent: "no-such-agent"}, "unknown_agent"},
		{host.NewArgs{Argv: []string{"true"}, Cwd: "~/missing"}, "bad_folder"},
		{host.NewArgs{Argv: []string{"no-such-program-here"}}, "agent_missing"},
		{host.NewArgs{Argv: []string{"true"}, Cwd: "~", Worktree: true}, "not_a_repo"},
	} {
		_, err := host.New(c.args)
		var pe *proto.Error
		if !errors.As(err, &pe) || pe.Code != c.code || pe.Hint == "" {
			t.Errorf("new %+v = %v, want a %s error with a hint", c.args, err, c.code)
		}
	}
	// A program that cannot run: the holder reports why, and New says so.
	bad := filepath.Join(h.Home, "not-a-program")
	writeFile(t, bad, "this is not a script\n")
	if err := os.Chmod(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := host.New(host.NewArgs{Argv: []string{bad}})
	var pe *proto.Error
	if !errors.As(err, &pe) || pe.Code != "start_failed" || !strings.Contains(pe.Message, "exec format error") {
		t.Fatalf("new with a program that cannot run = %v", err)
	}
	list, _ := host.List()
	if len(list) != 1 || list[0].State != host.StateErrored || list[0].Alive {
		t.Fatalf("the session that failed to start: %+v", list)
	}
}

func TestDeleteRefusesWhatIsNotASession(t *testing.T) {
	hosttest.New(t)
	for _, id := range []string{"", "../etc", "UPPER", "nosuchid"} {
		_, err := host.Delete(id)
		var pe *proto.Error
		if !errors.As(err, &pe) || pe.Code != "not_found" {
			t.Errorf("delete %q = %v", id, err)
		}
	}
}
