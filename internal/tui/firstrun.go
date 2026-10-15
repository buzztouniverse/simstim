package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// gradient is the brand gradient over n cells, or nil with 16 colours,
// where the mark falls back to the brand slot.
func (m *Model) gradient(n int) []color.Color {
	if !m.richColor || n <= 0 {
		return nil
	}
	return lipgloss.Blend1D(n, brandStops(m.dark)...)
}

// When the welcome's parts arrive, after the ribbon starts to open.
const (
	taglineAt = 700 * time.Millisecond
	keysAt    = 950 * time.Millisecond
)

// viewFirstRun is the screen with no hosts (docs/ux.md 2.1): the ribbon in
// the middle, the name decoding under it, the tagline, and the keys. Errors
// and notices go below.
func (m *Model) viewFirstRun() string {
	add := "add a host"
	if n := len(m.sshHosts); n > 0 {
		add += fmt.Sprintf(" (%d in your SSH config)", n)
	}
	center := func(s string) string { return strings.Repeat(" ", max((m.width-cells(s))/2, 0)) + s }
	var lines []string
	for _, l := range m.markLines() {
		lines = append(lines, center(l))
	}
	lines = append(lines, "", center(m.welcomeTitle()))
	if m.since(m.epoch) >= taglineAt || !m.motion {
		lines = append(lines, center(sFaint.Render(brand.Tagline)))
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, "")
	if m.since(m.epoch) >= keysAt || !m.motion {
		lines = append(lines, center(welcomeLine([]hint{{"a", add}, {"ctrl+o", "commands"}, {"?", "keys"}}, m.width-4)))
	} else {
		lines = append(lines, "")
	}
	var below []string
	room := min(m.width-4, 72)
	if m.loadErr != nil {
		msg, _ := errText(m.loadErr)
		below = append(below, errorBlock(msg, "Fix or move the hosts file, then start again.", room)...)
	}
	for _, n := range m.noticeLines(room) {
		below = append(below, n.s)
	}
	if len(below) > 0 {
		left := strings.Repeat(" ", max((m.width-room)/2, 0))
		lines = append(lines, "")
		for _, l := range below {
			lines = append(lines, left+l)
		}
	}
	top := max((m.height-len(lines))/2, 0)
	out := make([]string, 0, m.height)
	for range top {
		out = append(out, "")
	}
	out = append(out, lines...)
	for len(out) < m.height {
		out = append(out, "")
	}
	return strings.Join(out[:m.height], "\n")
}
