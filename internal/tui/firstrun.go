package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// gradient is the brand gradient over n cells, or nil with 16 colours,
// where the mark falls back to the brand slot.
func (m *Model) gradient(n int) []color.Color {
	if !m.richColor || n <= 0 {
		return nil
	}
	return lipgloss.Blend1D(n, brandStops(m.dark)...)
}

// viewFirstRun is the screen with no hosts (docs/ux.md 2.1), laid out like
// Amp's welcome: the mark on the left and a few short lines beside it, or
// the mark above them in a narrow terminal. Errors, notices and the
// shortcuts go below.
func (m *Model) viewFirstRun() string {
	add := "to add a host"
	if n := len(m.sshHosts); n > 0 {
		add += fmt.Sprintf(" (%s in your SSH config)", plural(n, "host", "hosts"))
	}
	text := []string{
		m.welcomeTitle(),
		"",
		welcomeLine([]hint{{"a", add}}, 60),
		welcomeLine([]hint{{"ctrl+o", "for commands"}, {"?", "for shortcuts"}}, 60),
	}
	textW := 0
	for _, l := range text {
		textW = max(textW, cells(l))
	}
	mark := m.markLines()
	var block []string
	left := 0
	if m.width >= markW+4+textW+4 {
		// Side by side, the text centred on the mark.
		left = max((m.width-(markW+4+textW))/2, 0)
		off := (len(mark) - len(text)) / 2
		for i := range max(len(mark), off+len(text)) {
			l := strings.Repeat(" ", markW)
			if i < len(mark) {
				l = mark[i]
			}
			if j := i - off; j >= 0 && j < len(text) {
				l += "    " + text[j]
			}
			block = append(block, strings.Repeat(" ", left)+l)
		}
		left += markW + 4
	} else {
		pad := strings.Repeat(" ", max((m.width-markW)/2, 0))
		for _, l := range mark {
			block = append(block, pad+l)
		}
		block = append(block, "")
		left = max((m.width-textW)/2, 0)
		for _, l := range text {
			block = append(block, strings.Repeat(" ", left)+l)
		}
	}
	var below []string
	room := max(m.width-left-2, 20)
	if m.loadErr != nil {
		msg, _ := errText(m.loadErr)
		below = append(below, errorBlock(msg, "Fix or move the hosts file, then start again.", room)...)
	}
	for _, n := range m.noticeLines(room) {
		below = append(below, n.s)
	}
	if m.showHelp {
		below = append(below, m.helpLines(min(room, 40))...)
	}
	lines := block
	if len(below) > 0 {
		lines = append(lines, "")
		for _, l := range below {
			lines = append(lines, strings.Repeat(" ", left)+l)
		}
	}
	top := max((m.height-len(block))/2-1, 0)
	if top+len(lines) > m.height {
		top = max(m.height-len(lines), 0)
	}
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
