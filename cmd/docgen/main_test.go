package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iAmLakshya/simstim/internal/brand"
)

func TestPageName(t *testing.T) {
	for in, want := range map[string]string{
		brand.Name:                  "index.md",
		brand.Name + " host":        "host.md",
		brand.Name + " host add":    "host_add.md",
		brand.Name + " config path": "config_path.md",
	} {
		if got := pageName(in); got != want {
			t.Errorf("pageName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The generator writes every page, leaves no page of a removed command, and
// gives the same bytes on a second run.
func TestRunIsStable(t *testing.T) {
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	stale := filepath.Join(dir, "cli", "gone.md")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	man, comp := filepath.Join(dir, "man"), filepath.Join(dir, "comp")
	if err := run(dir, man, comp); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("the page of a removed command was kept")
	}
	first := snapshot(t, dir)
	if err := run(dir, "", ""); err != nil {
		t.Fatal(err)
	}
	if second := snapshot(t, dir); second != first {
		t.Error("a second run changed the pages")
	}
	for _, f := range []string{"config.md", "keymap.md", "agents.md", "cli/index.md", "cli/host_add.md"} {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil || !strings.Contains(string(b), "Do not edit by hand") {
			t.Errorf("%s: missing or not marked generated", f)
		}
	}
	if m, _ := filepath.Glob(filepath.Join(man, "*.1")); len(m) < 10 {
		t.Errorf("%d man pages", len(m))
	}
	if c, _ := filepath.Glob(filepath.Join(comp, "*")); len(c) != 4 {
		t.Errorf("%d completion scripts, want 4", len(c))
	}
}

func snapshot(t *testing.T, dir string) string {
	t.Helper()
	var b strings.Builder
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".md") {
			c, _ := os.ReadFile(p)
			b.WriteString(p + "\n" + string(c))
		}
		return nil
	})
	return b.String()
}
