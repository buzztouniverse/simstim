package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/iAmLakshya/simstim/internal/client"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

// screen is what the terminal shows, without colours or styles.
func screen(m *Model) string {
	lines := strings.Split(ansi.Strip(m.render()), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n") + "\n"
}

func golden(t *testing.T, name string, m *Model) {
	t.Helper()
	got := screen(m)
	for i, l := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if w := ansi.StringWidth(l); w > m.width {
			t.Errorf("%s: line %d is %d cells wide, more than %d", name, i+1, w, m.width)
		}
	}
	if n := strings.Count(got, "\n"); n != m.height {
		t.Errorf("%s: %d lines, want %d", name, n, m.height)
	}
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test ./internal/tui -update to write it)", err)
	}
	if got != string(want) {
		t.Errorf("%s differs from %s:\n--- got\n%s--- want\n%s", name, path, got, want)
	}
}

// home is the lab, with atlas-dev selected and its preview loaded.
func home(t *testing.T, w, h int) *harness {
	hs := newHarness(t, labBackend(), w, h)
	hs.key("down") // atlas-dev
	hs.fire(isPreviewTick)
	return hs
}

func isPreviewTick(msg tea.Msg) bool { _, ok := msg.(previewTickMsg); return ok }

var sizes = []struct{ w, h int }{{80, 24}, {120, 36}}

func TestGolden(t *testing.T) {
	views := []struct {
		name  string
		setup func(t *testing.T, w, h int) *Model
	}{
		{"first_run", func(t *testing.T, w, h int) *Model {
			return newHarness(t, newFakeBackend(), w, h).m
		}},
		{"home", func(t *testing.T, w, h int) *Model { return home(t, w, h).m }},
		{"home_sessions", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("tab", "down")
			hs.fire(isPreviewTick)
			return hs.m
		}},
		{"all_hosts", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("up", "tab", "space")
			return hs.m
		}},
		{"offline", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("down")
			return hs.m
		}},
		{"new_session", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("enter", "tab")
			hs.typeText("/src/app")
			hs.key("tab")
			hs.typeText("auth")
			return hs.m
		}},
		{"new_session_error", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.be.state["atlas-dev"].newErr = &client.UserError{Msg: "aider is not installed on this host", Hint: "Install it on the host, then try again."}
			hs.key("enter", "enter")
			return hs.m
		}},
		{"delete", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("tab", "down", "d")
			return hs.m
		}},
		{"rename", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("tab", "r", "ctrl+u")
			hs.typeText("auth-v2")
			return hs.m
		}},
		{"filter_empty", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("tab", "/")
			hs.typeText("xyz")
			return hs.m
		}},
		{"palette", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("ctrl+o")
			hs.typeText("ses")
			return hs.m
		}},
		{"help", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("?")
			return hs.m
		}},
		{"details", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("i")
			return hs.m
		}},
		{"toasts", func(t *testing.T, w, h int) *Model {
			hs := home(t, w, h)
			hs.key("tab", "y")
			hs.m.toastErr(&client.UserError{Msg: "cannot reach aws-gpu: timed out", Hint: "ssh aws-gpu"})
			return hs.m
		}},
		{"wizard_where", func(t *testing.T, w, h int) *Model {
			be := newFakeBackend()
			be.known = []string{"atlas-dev", "gcp-dev", "hetzner-1"}
			hs := newHarness(t, be, w, h)
			hs.key("a")
			hs.typeText("het")
			return hs.m
		}},
		{"wizard_test_failed", func(t *testing.T, w, h int) *Model {
			be := newFakeBackend()
			be.probeErr = &client.UserError{Msg: "cannot reach box: Permission denied (publickey).", Hint: "ssh box   (check that this works first)"}
			hs := newHarness(t, be, w, h)
			hs.key("a")
			hs.typeText("box")
			hs.key("enter")
			return hs.m
		}},
		{"wizard_install", func(t *testing.T, w, h int) *Model {
			be := newFakeBackend()
			be.probe = client.Platform{OS: "linux", Arch: "arm64"}
			hs := newHarness(t, be, w, h)
			hs.key("a")
			hs.typeText("pi@raspberry")
			hs.key("enter", "enter")
			return hs.m
		}},
		{"wizard_ready", func(t *testing.T, w, h int) *Model {
			be := newFakeBackend()
			be.probe = client.Platform{OS: "linux", Arch: "arm64"}
			hs := newHarness(t, be, w, h)
			hs.key("a")
			hs.typeText("pi@raspberry")
			hs.key("enter", "enter", "enter")
			return hs.m
		}},
	}
	for _, v := range views {
		for _, s := range sizes {
			name := fmt.Sprintf("%s_%dx%d", v.name, s.w, s.h)
			t.Run(name, func(t *testing.T) {
				golden(t, name, v.setup(t, s.w, s.h))
			})
		}
	}
}
