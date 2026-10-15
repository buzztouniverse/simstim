package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iAmLakshya/simstim/internal/brand"
)

func TestPageName(t *testing.T) {
	for in, want := range map[string]string{
		brand.Name:                  "_index.md",
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
	for _, f := range []string{"config.md", "keymap.md", "agents.md", "cli/_index.md", "cli/host_add.md"} {
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

// The site hosts every page in docs/ and the community files at the root,
// and maps each one's path in the repository to its page, so links between
// them stay on the site.
func TestSitePages(t *testing.T) {
	if _, err := os.Stat("go.mod"); err != nil {
		if err := os.Chdir("../.."); err != nil {
			t.Fatal(err)
		}
	}
	out := t.TempDir()
	if err := sitePages("docs", out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "data", "pages.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pages map[string]string
	if err := json.Unmarshal(b, &pages); err != nil {
		t.Fatal(err)
	}
	for src, want := range map[string]string{
		"README.md":                    "/",
		"docs/README.md":               "/docs",
		"docs/how-to/README.md":        "/how-to",
		"docs/how-to":                  "/how-to",
		"docs/how-to/install.md":       "/how-to/install",
		"docs/reference/_index.md":     "/reference",
		"CHANGELOG.md":                 "/project/changelog",
		"CONTRIBUTING.md":              "/project/contributing",
		"examples/providers/README.md": "/reference/provider-scripts",
	} {
		if got := pages[src]; got != want {
			t.Errorf("pages[%q] = %q, want %q", src, got, want)
		}
	}
	for _, f := range []string{"content/_index.md", "content/docs.md", "content/project/_index.md", "content/project/changelog.md", "data/brand.json", "data/agents.json"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	page, _ := os.ReadFile(filepath.Join(out, "content", "how-to", "install.md"))
	if !strings.Contains(string(page), "srcPath: \"docs/how-to/install.md\"\n") {
		t.Errorf("install.md has no srcPath:\n%.200s", page)
	}
}
