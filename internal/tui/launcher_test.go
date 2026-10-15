package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/client"
)

// The launcher opens where the last run left off, with the sessions it saw,
// before any host has answered.
func TestMemoryOpensWhereYouLeftOff(t *testing.T) {
	be := labBackend()
	be.mem = memory{
		Sessions: map[string][]client.Session{"gcp-dev": be.state["gcp-dev"].sessions},
		Last:     &lastSession{Host: "gcp-dev", ID: "g1", Name: "bench"},
	}
	m := New(context.Background(), be, Options{Now: func() time.Time { return now }})
	if m.screen != scrSessions || m.cur != "gcp-dev" {
		t.Fatalf("opened on %v %q, want gcp-dev's sessions", m.screen, m.cur)
	}
	if s := m.selected(); s == nil || s.ID != "g1" {
		t.Fatalf("selected %v before any host answered, want the last session", s)
	}
	m.width, m.height = 80, 24
	if out := ansi.Strip(m.render()); !strings.Contains(out, "bench") || !strings.Contains(out, "last seen before this run") {
		t.Fatalf("the remembered sessions do not show at once:\n%s", out)
	}

	hs := newHarness(t, be, 80, 24)
	if hs.m.sel != (sessKey{"gcp-dev", "g1"}) {
		t.Fatalf("after the host answered the selection moved to %v", hs.m.sel)
	}
	hs.key("enter")
	if len(be.attached) != 1 || be.attached[0].Session != "g1" {
		t.Fatal("Enter did not take you straight back in")
	}
}

func TestAttachAndCloseRemember(t *testing.T) {
	hs := sessions(t, 80, 24)
	hs.key("enter")
	if l := hs.be.mem.Last; l == nil || l.Host != "atlas-dev" || l.ID != "s4" {
		t.Fatalf("the last session is %+v", l)
	}
	hs.m.Close()
	if n := len(hs.be.mem.Sessions["atlas-dev"]); n != 4 {
		t.Fatalf("closing saved %d sessions of atlas-dev, want 4", n)
	}
	if _, ok := hs.be.mem.Sessions["aws-gpu"]; ok {
		t.Fatal("a host that never answered was saved with sessions")
	}
}

func TestQuickOpen(t *testing.T) {
	hs := home(t, 80, 24)
	hs.key("3")
	if hs.m.screen != scrSessions || hs.m.cur != "gcp-dev" {
		t.Fatalf("3 opened %q", hs.m.cur)
	}
	hs.key("1")
	if len(hs.be.attached) != 1 || hs.be.attached[0].Session != "g1" {
		t.Fatalf("1 attached %+v", hs.be.attached)
	}
	if !strings.Contains(plainScreen(hs.m), "1 ≈ bench") {
		t.Fatal("rows do not show their numbers")
	}
}

func TestMouse(t *testing.T) {
	hs := sessions(t, 80, 24)
	hs.m.render()
	g := hs.m.geo
	click := func(x, y int) { hs.send(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}) }
	click(g.x+10, g.listY+1)
	if hs.m.sel.id != "s3" || len(hs.be.attached) != 0 {
		t.Fatalf("a click on the second row selected %v", hs.m.sel)
	}
	click(g.x+10, g.listY+1)
	if len(hs.be.attached) != 1 || hs.be.attached[0].Session != "s3" {
		t.Fatal("a click on the selected row did not attach")
	}
	hs.send(tea.MouseWheelMsg{X: g.x + 10, Y: g.listY, Button: tea.MouseWheelDown})
	if hs.m.sel.id != "s2" {
		t.Fatalf("the wheel moved to %v", hs.m.sel)
	}
	hs.key("ctrl+o")
	hs.m.render()
	click(hs.m.modalGeo.x+10, hs.m.modalGeo.y+4) // the first command: session new
	if _, ok := hs.m.modal.(*newSession); !ok {
		t.Fatalf("a click on a palette row opened %T", hs.m.modal)
	}
}

func TestTitleCountsTheSessionsThatNeedYou(t *testing.T) {
	hs := home(t, 80, 24)
	if got := hs.m.View().WindowTitle; !strings.HasPrefix(got, "● 1 · ") {
		t.Fatalf("title %q", got)
	}
}

func TestNotifyWhenNobodyIsLooking(t *testing.T) {
	hs := sessions(t, 80, 24)
	hs.send(tea.BlurMsg{})
	h := hs.be.state["atlas-dev"]
	hs.be.mu.Lock()
	h.sessions[2].State, h.sessions[2].Reason = api.StateNeedsInput, "Run the tests?"
	hs.be.mu.Unlock()
	notes := func() []string {
		var out []string
		for _, r := range hs.raw {
			if strings.Contains(r, "\x1b]777;notify;") {
				out = append(out, r)
			}
		}
		return out
	}
	hs.fire(func(m tea.Msg) bool { _, ok := m.(refreshTickMsg); return ok })
	if n := notes(); len(n) != 1 || !strings.Contains(n[0], "api-tests needs you: Run the tests?") {
		t.Fatalf("notifications %q", n)
	}
	hs.send(tea.FocusMsg{})
	hs.be.mu.Lock()
	h.sessions[1].State = api.StateNeedsInput
	hs.be.mu.Unlock()
	hs.fire(func(m tea.Msg) bool { _, ok := m.(refreshTickMsg); return ok })
	if len(notes()) != 1 {
		t.Fatal("a notification went out while the terminal had focus")
	}
}

func TestFilterHighlights(t *testing.T) {
	if got := fuzzyPos("atr", "auth-refactor"); got == nil || !got[0] || !got[2] || !got[5] {
		t.Fatalf("fuzzyPos = %v", got)
	}
	if fuzzyPos("zz", "auth") != nil || fuzzyPos("", "auth") != nil {
		t.Fatal("no match, or no query, should mark nothing")
	}
}

func TestTermColors(t *testing.T) {
	var tc termColors
	tc.onOSC("\x1b]4;3;rgb:f9f9/afaf/4f4f\x1b\\")
	c, ok := tc.resolve(lipgloss.Yellow, true)
	if !ok || c != (rgb{0xf9, 0xaf, 0x4f}) {
		t.Fatalf("slot 3 = %v %v", c, ok)
	}
	if _, ok := tc.resolve(nil, true); ok {
		t.Fatal("the default colour resolved before the terminal said it")
	}
	if _, ok := tc.resolve(lipgloss.Blue, true); ok {
		t.Fatal("an unknown slot resolved")
	}
}

func TestGlowTravels(t *testing.T) {
	m := &Model{}
	c := lipgloss.NewCanvas(20, 5)
	b := box(20, 5, "", "", "", nil, toneNormal)
	drawn := func() *lipgloss.Canvas {
		c = lipgloss.NewCanvas(20, 5)
		uvDraw(c, b)
		return c
	}
	m.applyGlow(drawn(), glow{w: 20, h: 5, pos: 0, strength: 1})
	head, far := c.CellAt(0, 0).Style, c.CellAt(10, 4).Style
	if head.Fg != lipgloss.BrightWhite || far.Attrs == 0 {
		t.Fatalf("without blends the head is bright and the rest faint: head %+v, far %+v", head, far)
	}
	m.applyGlow(drawn(), glow{w: 20, h: 5, pos: 0.5, strength: 1})
	if c.CellAt(0, 0).Style.Fg == lipgloss.BrightWhite {
		t.Fatal("the glow did not travel")
	}
}

func TestLifeFrames(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range lifeFrames {
		r := []rune(f)
		if len(r) != 1 || r[0] < 0x2801 || r[0] > 0x28FF {
			t.Fatalf("frame %q is not a braille cell with dots", f)
		}
		seen[f] = true
	}
	if len(seen) < 10 {
		t.Fatalf("only %d different frames", len(seen))
	}
}
