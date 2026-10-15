package host

import (
	"io"
	"strings"

	"github.com/charmbracelet/x/vt"

	"github.com/iamlakshya/simstim/internal/agents"
)

// screenModel keeps a rendered copy of the session's screen, only to read its
// text for status rules. If the emulator ever panics, the model turns itself
// off: status falls back to the other signals, and the agent is never touched.
type screenModel struct {
	e     *vt.Emulator
	title string
	dirty bool
	off   bool
}

func newScreenModel(cols, rows int) *screenModel {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	m := &screenModel{e: vt.NewEmulator(cols, rows)}
	m.e.SetScrollbackSize(0)
	// The emulator writes replies to queries into a pipe: drain and drop them.
	go func() { _, _ = io.Copy(io.Discard, m.e) }()
	return m
}

func (m *screenModel) feed(p []byte) {
	if m == nil || m.off {
		return
	}
	defer func() {
		if recover() != nil {
			m.off = true
		}
	}()
	_, _ = m.e.Write(p)
	m.dirty = true
}

func (m *screenModel) resize(cols, rows int) {
	if m == nil || m.off || cols <= 0 || rows <= 0 {
		return
	}
	defer func() {
		if recover() != nil {
			m.off = true
		}
	}()
	m.e.Resize(cols, rows)
	m.dirty = true
}

// altScreen is true while the program draws on the alternate screen.
func (m *screenModel) altScreen() bool {
	if m == nil || m.off {
		return false
	}
	defer func() { _ = recover() }()
	return m.e.IsAltScreen()
}

// screen returns the visible text, one string per row.
func (m *screenModel) screen() (s agents.Screen) {
	if m == nil || m.off {
		return s
	}
	defer func() {
		if recover() != nil {
			m.off = true
		}
	}()
	s.Title = m.title
	w, h := m.e.Width(), m.e.Height()
	var b strings.Builder
	for y := 0; y < h; y++ {
		b.Reset()
		for x := 0; x < w; x++ {
			if c := m.e.CellAt(x, y); c != nil && c.Content != "" {
				b.WriteString(c.Content)
			} else if c == nil || c.Width != 0 {
				b.WriteByte(' ')
			}
		}
		s.Lines = append(s.Lines, strings.TrimRight(b.String(), " "))
	}
	return s
}
