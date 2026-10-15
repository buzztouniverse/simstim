package host_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/host/hosttest"
	"github.com/iAmLakshya/simstim/internal/proto"
)

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
