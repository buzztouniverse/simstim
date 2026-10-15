package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/iAmLakshya/simstim/internal/client"
)

// onKey routes a key: to the dialog on top, then an inline rename, then a
// filter being typed, then the moves and the command registry.
func (m *Model) onKey(k tea.KeyPressMsg) tea.Cmd {
	ks := k.String()
	if ks == "ctrl+c" {
		return m.ctrlC()
	}
	if m.modal != nil {
		return m.modal.key(m, k)
	}
	if m.rename != nil {
		return m.renameKey(k)
	}
	if f := m.typingFilter(); f != nil {
		return m.filterKey(f, k)
	}
	if len(m.hosts) > 0 {
		if m.navKey(ks) {
			return nil
		}
		if ks == "enter" && m.focus == paneHosts && m.selectedHost() == nil {
			m.focus = paneSessions
			return nil
		}
	}
	for _, c := range commands {
		if c.matches(m, ks) {
			return m.runCommand(c)
		}
	}
	return nil
}

// navKey moves around the home screen. It reports whether it used the key.
func (m *Model) navKey(ks string) bool {
	switch ks {
	case "up", "k", "down", "j", "home", "end", "pgup", "pgdown":
		d := map[string]int{"up": -1, "k": -1, "down": 1, "j": 1, "home": -1 << 20, "end": 1 << 20, "pgup": -10, "pgdown": 10}[ks]
		if m.focus == paneHosts {
			m.moveHost(d)
		} else if d == 1 || d == -1 {
			m.moveSel(d)
		} else {
			m.jumpSel(d)
		}
	case "tab", "shift+tab":
		m.focus = 1 - m.focus
	case "left", "h":
		m.focus = paneHosts
	case "right", "l":
		m.focus = paneSessions
	case "esc":
		if m.sessFilter.query() != "" || m.hostFilter.query() != "" {
			m.keepHost(func() { m.sessFilter, m.hostFilter = filter{}, filter{} })
			m.syncSel()
		}
	default:
		return false
	}
	return true
}

func (m *Model) jumpSel(d int) {
	rows := m.rows()
	target := min(max(m.selIdx+d, 0), len(rows)-1)
	if i := nearestSession(rows, target); i >= 0 {
		m.selIdx, m.sel, m.autoSel = i, rows[i].key(), false
		m.scrollTo(len(rows))
	}
}

// ctrlC quits on the second press within a second.
func (m *Model) ctrlC() tea.Cmd {
	now := m.now()
	if !m.quitArmed.IsZero() && now.Sub(m.quitArmed) <= quitWindow {
		return tea.Quit
	}
	m.quitArmed = now
	return m.toast(toastInfo, "Press ctrl+c again to quit. Sessions keep running.")
}

func (m *Model) typingFilter() *filter {
	if m.hostFilter.typing {
		return &m.hostFilter
	}
	if m.sessFilter.typing {
		return &m.sessFilter
	}
	return nil
}

func (m *Model) startFilter() {
	f := &m.sessFilter
	if m.focus == paneHosts {
		f = &m.hostFilter
	}
	f.typing = true
}

// keepHost runs fn, which changes the host list, and keeps the same host
// selected.
func (m *Model) keepHost(fn func()) {
	h := m.selectedHost()
	fn()
	if h != nil {
		m.hostSel = indexOf(m.visibleHosts(), h) + 1
	}
}

func (m *Model) filterKey(f *filter, k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		m.keepHost(func() { *f = filter{} })
	case "enter":
		f.typing = false
	case "up", "down":
		f.typing = false
		m.navKey(k.String())
		return nil
	default:
		f.in.key(k)
	}
	if f == &m.hostFilter {
		m.hostSel = min(m.hostSel, len(m.visibleHosts()))
		if m.hostFilter.query() != "" && len(m.visibleHosts()) > 0 {
			m.hostSel = 1
		}
	}
	m.autoSel = true
	m.syncSel()
	return nil
}

// onPaste handles a paste at home: into a field being typed, or, if it is a
// session ID, a jump to that session.
func (m *Model) onPaste(text string) tea.Cmd {
	switch {
	case m.modal != nil:
		if p, ok := m.modal.(interface{ paste(string) }); ok {
			p.paste(text)
		}
		return nil
	case m.rename != nil:
		m.rename.in.insert(text)
		return nil
	case m.typingFilter() != nil:
		m.typingFilter().in.insert(text)
		m.syncSel()
		return nil
	}
	id := strings.TrimSpace(text)
	if _, ref, err := client.SplitTarget(id); err == nil {
		id = ref
	}
	for _, h := range m.hosts {
		for _, s := range h.sessions {
			if s.ID == id || (len(id) >= 4 && strings.HasPrefix(s.ID, id)) {
				m.hostSel, m.focus = 0, paneSessions
				m.sessFilter, m.hostFilter = filter{}, filter{}
				m.sel, m.autoSel = sessKey{h.name(), s.ID}, false
				m.syncSel()
				return nil
			}
		}
	}
	return m.toast(toastInfo, fmt.Sprintf("No session with the ID %s.", trunc(id, 24)))
}

// Rename.

func (m *Model) startRename() tea.Cmd {
	h, s := m.selectedPair()
	if s == nil {
		return nil
	}
	m.focus = paneSessions
	m.rename = &renameState{key: sessKey{h.name(), s.ID}, old: s.Name, in: newInput(s.Name, "a name")}
	return nil
}

func (m *Model) renameKey(k tea.KeyPressMsg) tea.Cmd {
	r := m.rename
	if r.busy {
		return nil
	}
	switch k.String() {
	case "esc":
		m.rename = nil
		return nil
	case "enter":
		name := strings.TrimSpace(r.in.value())
		h := m.hostByName(r.key.host)
		if name == "" || name == r.old || h == nil {
			m.rename = nil
			return nil
		}
		r.busy = true
		h.changes++
		for i := range h.sessions {
			if h.sessions[i].ID == r.key.id {
				h.sessions[i].Name = name // show it now; the next list confirms it
			}
		}
		id, old := r.key.id, r.old
		return m.op(h, fmt.Sprintf("Renamed %s to %s on %s.", old, name, h.name()), func(ctx context.Context, c Conn) error {
			return c.RenameSession(ctx, id, name)
		})
	}
	r.in.key(k)
	return nil
}
