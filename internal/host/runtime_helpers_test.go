package host_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/host/hosttest"
)

// start starts a session that runs argv in the home folder.
func start(t *testing.T, name string, argv ...string) *host.SessionInfo {
	t.Helper()
	s, err := host.New(host.NewArgs{Name: name, Cwd: "~", Argv: argv, Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("new %s: %v", name, err)
	}
	return s
}

// peek returns the newest output of a session.
func peek(t *testing.T, id string) string {
	t.Helper()
	raw, err := host.HolderRequest(id, "peek", map[string]int{"bytes": 64 << 10}, 3*time.Second)
	if err != nil {
		t.Fatalf("peek: %v", err)
	}
	var r struct {
		Data []byte `json:"data"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return string(r.Data)
}

// waitOutput waits until a session's output holds want.
func waitOutput(t *testing.T, id, want string) {
	t.Helper()
	hosttest.WaitFor(t, 10*time.Second, "output "+want, func() bool { return strings.Contains(peek(t, id), want) })
}

// live returns the holder's own copy of a session's manifest, which is
// current: the file is saved only once a second.
func live(t *testing.T, id string) *host.Manifest {
	t.Helper()
	raw, err := host.HolderRequest(id, "info", nil, 3*time.Second)
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	var m host.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return &m
}

// waitState waits until a session's manifest has the state.
func waitState(t *testing.T, id, state string) *host.Manifest {
	t.Helper()
	var m *host.Manifest
	hosttest.WaitFor(t, 10*time.Second, "state "+state, func() bool {
		var err error
		m, err = host.LoadManifest(id)
		return err == nil && m.Status.State == state
	})
	return m
}

// writeFile writes a file, making its folder.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
