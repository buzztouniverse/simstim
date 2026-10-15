package host

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/paths"
)

func TestPruneKeepsWhatIsInUse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	bin := paths.BinDir()
	for _, v := range []string{"old-1", "old-2", "running", "in-unit", "now", ".upload.1"} {
		if err := os.MkdirAll(filepath.Join(bin, v), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("now", filepath.Join(bin, "current")); err != nil {
		t.Fatal(err)
	}
	// Every version is old, oldest first, except "fresh", installed a
	// minute ago; "new-1" and "new-2" are the newest old ones.
	for _, v := range []string{"new-1", "new-2"} {
		if err := os.MkdirAll(filepath.Join(bin, v), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for i, v := range []string{"old-1", "old-2", "running", "in-unit", "now", "new-1", "new-2"} {
		at := time.Now().Add(-3*pruneGrace + time.Duration(i)*time.Minute)
		if err := os.Chtimes(filepath.Join(bin, v), at, at); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(bin, "fresh"), 0o700); err != nil {
		t.Fatal(err)
	}
	// A holder that runs from "running", and a process of something else.
	procRoot = filepath.Join(home, "proc")
	t.Cleanup(func() { procRoot = "/proc" })
	for pid, exe := range map[string]string{
		"100": filepath.Join(bin, "running", brand.HostBinary) + " (deleted)",
		"200": "/usr/bin/bash",
	} {
		if err := os.MkdirAll(filepath.Join(procRoot, pid), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(exe, filepath.Join(procRoot, pid, "exe")); err != nil {
			t.Fatal(err)
		}
	}
	// A unit file that names "in-unit".
	if err := os.MkdirAll(unitDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	unit := "[Service]\nExecStart=" + filepath.Join(bin, "in-unit", brand.HostBinary) + " idle check\n"
	if err := os.WriteFile(filepath.Join(unitDir(), "idle.service"), []byte(unit), 0o600); err != nil {
		t.Fatal(err)
	}

	removed, err := Prune()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"old-2", "old-1"}; !reflect.DeepEqual(removed, want) { // newest first
		t.Errorf("removed %q, want %q", removed, want)
	}
	for _, v := range []string{"running", "in-unit", "now", "fresh", "new-1", "new-2", ".upload.1", "current"} {
		if _, err := os.Lstat(filepath.Join(bin, v)); err != nil {
			t.Errorf("%s was removed: %v", v, err)
		}
	}
}

func TestPruneWithNoHelpers(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if removed, err := Prune(); err != nil || len(removed) != 0 {
		t.Fatalf("%q %v", removed, err)
	}
}
