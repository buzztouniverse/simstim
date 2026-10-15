package host

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iamlakshya/simstim/internal/paths"
)

// procRoot is /proc; tests point it at a fake one.
var procRoot = "/proc"

// pruneGrace keeps a version that was installed this recently, so that a
// client installing at the same moment as another can still start it.
const pruneGrace = time.Hour

// unitDir is where systemd keeps the user's own unit files.
func unitDir() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, "systemd", "user")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config", "systemd", "user")
}

// Prune removes the helper versions that nothing uses any more (see
// oldHelpers), and returns their names. Each install leaves a new version
// behind, so without this they pile up on a host. The client runs it after
// each install; host doctor --fix offers the same.
func Prune() ([]string, error) {
	old := oldHelpers()
	names := make([]string, len(old))
	for i, p := range old {
		names[i] = filepath.Base(p)
	}
	return names, removeAll(old, os.RemoveAll)
}

// oldHelpers is every helper version folder that can go: not one of the
// newest keepHelpers, not in use (helpersInUse), and not installed within
// pruneGrace. Newest first.
func oldHelpers() []string {
	used := helpersInUse()
	var old []string
	for i, d := range helperDirs() {
		if i < keepHelpers || used[d.Name()] || time.Since(modTime(d)) < pruneGrace {
			continue
		}
		old = append(old, filepath.Join(paths.BinDir(), d.Name()))
	}
	return old
}

// helpersInUse names the helper versions that something uses: the current
// link, this helper, the holder of each running session, any process of the
// user that runs a binary from a version folder (a holder, or its shadow
// terminal, which restarts from the same folder), and any unit file that
// names one.
func helpersInUse() map[string]bool {
	bin := paths.BinDir()
	used := map[string]bool{}
	if t, err := os.Readlink(filepath.Join(bin, "current")); err == nil {
		used[filepath.Base(t)] = true
	}
	if v := ownVersion(); v != "" {
		used[v] = true
	}
	mark := func(path string) {
		if v := versionOf(bin, strings.TrimSuffix(path, " (deleted)")); v != "" {
			used[v] = true
		}
	}
	entries, _ := os.ReadDir(paths.SessionsDir())
	for _, e := range entries {
		if m, err := LoadManifest(e.Name()); err == nil && alive(e.Name()) {
			mark(m.HolderBinary)
		}
	}
	procs, _ := os.ReadDir(procRoot)
	for _, p := range procs {
		if exe, err := os.Readlink(filepath.Join(procRoot, p.Name(), "exe")); err == nil {
			mark(exe)
		}
	}
	files, _ := os.ReadDir(unitDir())
	prefix := []byte(bin + "/")
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(unitDir(), f.Name()))
		if err != nil {
			continue
		}
		for _, field := range bytes.Fields(b) {
			if i := bytes.Index(field, prefix); i >= 0 {
				mark(string(field[i:]))
			}
		}
	}
	return used
}

// versionOf is the version folder that path lies in under bin, or "".
func versionOf(bin, path string) string {
	rest, ok := strings.CutPrefix(path, bin+"/")
	if !ok {
		return ""
	}
	v, _, _ := strings.Cut(rest, "/")
	return v
}
