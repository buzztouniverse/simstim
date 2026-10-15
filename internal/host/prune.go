package host

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/iAmLakshya/simstim/internal/paths"
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

// Prune removes the helper versions that nothing uses any more, and returns
// their names. A version stays when it is the current one, when a process
// of this user runs from it (a session's holder, or its shadow terminal,
// which restarts from the same folder), or when one of the user's systemd
// unit files names it, or when it was installed within pruneGrace. Each
// install leaves a new version behind, so without
// this they pile up on a host.
func Prune() ([]string, error) {
	bin := paths.BinDir()
	entries, err := os.ReadDir(bin)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	keep := map[string]bool{}
	if cur, err := os.Readlink(filepath.Join(bin, "current")); err == nil {
		keep[filepath.Base(cur)] = true
	}
	for v := range versionsRunning(bin) {
		keep[v] = true
	}
	for v := range versionsInUnits(bin) {
		keep[v] = true
	}
	var removed []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || name == "current" || strings.HasPrefix(name, ".") || keep[name] {
			continue
		}
		if fi, err := e.Info(); err != nil || time.Since(fi.ModTime()) < pruneGrace {
			continue
		}
		if err := os.RemoveAll(filepath.Join(bin, name)); err != nil {
			return removed, err
		}
		removed = append(removed, name)
	}
	sort.Strings(removed)
	return removed, nil
}

// versionsRunning is the set of versions that this user's processes run
// from, read from /proc/<pid>/exe.
func versionsRunning(bin string) map[string]bool {
	out := map[string]bool{}
	procs, _ := os.ReadDir(procRoot)
	for _, p := range procs {
		exe, err := os.Readlink(filepath.Join(procRoot, p.Name(), "exe"))
		if err != nil {
			continue
		}
		if v := versionOf(bin, strings.TrimSuffix(exe, " (deleted)")); v != "" {
			out[v] = true
		}
	}
	return out
}

// versionsInUnits is the set of versions that the user's unit files name.
func versionsInUnits(bin string) map[string]bool {
	out := map[string]bool{}
	files, _ := os.ReadDir(unitDir())
	prefix := []byte(bin + "/")
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(unitDir(), f.Name()))
		if err != nil {
			continue
		}
		for _, field := range bytes.Fields(b) {
			if i := bytes.Index(field, prefix); i >= 0 {
				if v := versionOf(bin, string(field[i:])); v != "" {
					out[v] = true
				}
			}
		}
	}
	return out
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
