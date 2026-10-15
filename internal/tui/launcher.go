package tui

import (
	"fmt"
	"strings"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// The launcher (docs/ux.md 2.2 and 2.3) is one framed list in one column:
// the hosts, or the sessions on one host. Its title sits in the top border,
// its keys in the bottom border, and short notices appear under it.

// noticeRows are kept free under the box for notices, so the box never
// moves when one appears.
const noticeRows = 4

// boxW is the width of the launcher's box.
func (m *Model) boxW() int { return max(min(m.width-4, 96), min(m.width, 32)) }

// boxTop is the row the box starts on: a little way down, the same on
// every screen, so moving between them does not shift it.
func (m *Model) boxTop() int { return min(max(m.height/6, 1), 5) }

// cw is the width of a row inside the box.
func (m *Model) cw() int { return m.boxW() - 4 }

// listRoom is how many rows the list may take.
func (m *Model) listRoom() int {
	room := m.height - m.boxTop() - noticeRows - 4 // borders and a blank row at each end
	if m.activeFilter().shown() {
		room -= 2
	}
	if m.showHelp {
		room -= len(m.helpLines(m.cw())) + 2
	}
	return max(room, 1)
}

func (m *Model) activeFilter() *filter {
	if m.screen == scrHosts {
		return &m.hostFilter
	}
	return &m.sessFilter
}

// viewLauncher draws the box, and the notices to go under it.
func (m *Model) viewLauncher() (string, layer) {
	w, cw := m.boxW(), m.cw()
	var title, right string
	var body []string
	var hints []hint
	if m.screen == scrHosts {
		title, right, body, hints = m.hostsView(cw)
	} else {
		title, right, body, hints = m.sessionsView(cw)
	}
	switch {
	case m.rename != nil:
		hints = []hint{{"Enter", "rename"}, {"Esc", "cancel"}}
	case m.activeFilter().typing:
		hints = []hint{{"Enter", "keep filter"}, {"↑↓", "move"}, {"Esc", "clear"}}
	}
	var all []string
	if f := m.activeFilter(); f.shown() {
		all = append(all, "> "+f.in.view(cw-2, f.typing), "")
	}
	all = append(all, body...)
	if m.showHelp {
		all = append(all, "", strings.Repeat("─", cw))
		all = append(all, m.helpLines(cw)...)
	}
	all = append([]string{""}, append(all, "")...)
	b := strings.Split(box(w, len(all)+2, title, right, hintsFor(hints, w), all, toneNormal), "\n")

	x, top := max((m.width-w)/2, 0), m.boxTop()
	lines := make([]string, m.height)
	for i, l := range b {
		if top+i < m.height {
			lines[top+i] = strings.Repeat(" ", x) + l
		}
	}
	var notices layer
	if n := m.noticeLines(cw); len(n) > 0 && top+len(b)+1 < m.height {
		notices = layer{s: strings.Join(n, "\n"), x: x + 2, y: top + len(b) + 1}
	}
	return strings.Join(lines, "\n"), notices
}

// Hosts.

func (m *Model) hostsView(cw int) (title, right string, body []string, hints []hint) {
	title, right = "Hosts", sFaint.Render(plural(len(m.hosts), "host", "hosts"))
	hints = []hint{{"Enter", "open"}, {"n", "new session"}, {"a", "add host"}, {"Ctrl+O", "commands"}, {"?", "shortcuts"}}
	vh := m.visibleHosts()
	if len(vh) == 0 {
		return title, right, []string{sFaint.Render(trunc(fmt.Sprintf("Nothing matches '%s'.", m.hostFilter.query()), cw))}, hints
	}
	nameW := 8
	for _, h := range vh {
		nameW = max(nameW, cells(h.name()))
	}
	nameW = min(nameW, max(cw/3, 8))
	for i := m.hostTop; i < len(vh) && i < m.hostTop+m.listRoom(); i++ {
		body = append(body, m.hostRow(vh[i], i == m.hostSel, cw, nameW))
	}
	return title, right, body, hints
}

const hostRight = 8 // latency, or nothing

func (m *Model) hostRow(h *hostConn, selected bool, cw, nameW int) string {
	detailW := cw - 3 - nameW - 2 - 2 - hostRight - 1
	glyph := seg{glyphOnline, sSuccess}
	nameSt := sBold
	var detail []seg
	right := seg{fitRight(fmtLatency(h.latency), hostRight), sFaint}
	blank := seg{strings.Repeat(" ", hostRight), sPlain}
	switch h.status {
	case stConnecting:
		glyph, right = seg{m.spinner(), sKey}, blank
		detail = []seg{{"connecting…", sFaint}}
	case stOffline:
		glyph, nameSt, right = seg{glyphOffline, sFaint}, sFaint, blank
		detail = []seg{{"offline", sFaint}}
	case stFailed:
		glyph, right = seg{glyphFail, sDanger}, blank
		detail = []seg{{"not ready", sDanger}}
	default:
		detail = m.hostSummary(h)
	}
	used := 0
	for i := range detail {
		detail[i].s = trunc(detail[i].s, detailW-used)
		used += cells(detail[i].s)
	}
	segs := []seg{{" ", sPlain}, glyph, {" ", sPlain}, {fit(h.name(), nameW), nameSt}, {"  ", sPlain}}
	segs = append(segs, detail...)
	segs = append(segs, seg{strings.Repeat(" ", max(detailW-used, 0)), sPlain}, seg{"  ", sPlain}, right, seg{" ", sPlain})
	return listRow(selected, segs...)
}

// hostSummary is the faint line about a host's sessions; the count that
// needs the user stands out.
func (m *Model) hostSummary(h *hostConn) []seg {
	if !h.loaded {
		return []seg{{"…", sFaint}}
	}
	if len(h.sessions) == 0 {
		return []seg{{"no sessions", sFaint}}
	}
	out := []seg{{plural(len(h.sessions), "session", "sessions"), sFaint}}
	need := 0
	for _, s := range h.sessions {
		if s.State == "needs_input" {
			need++
		}
	}
	if need > 0 {
		out = append(out, seg{dot, sFaint}, seg{fmt.Sprintf("%d needs you", need), sWarning})
	}
	return out
}

// Sessions.

func (m *Model) sessionsView(cw int) (title, right string, body []string, hints []hint) {
	hints = []hint{{"Enter", "attach"}, {"n", "new"}, {"Space", "preview"}, {"Esc", "hosts"}, {"?", "shortcuts"}}
	h := m.selectedHost()
	if h == nil {
		title, right = "All sessions", sFaint.Render(plural(len(m.hosts), "host", "hosts"))
	} else {
		title, right = h.name(), m.hostState(h)
	}
	rows := m.rows()
	if sessionRows(rows) == 0 {
		return title, right, m.emptyState(cw), hints
	}
	cols := sessCols(cw)
	for i := m.sessTop; i < len(rows) && i < m.sessTop+m.listRoom(); i++ {
		body = append(body, m.sessRowView(rows[i], cols, i == m.selIdx))
	}
	return title, right, body, hints
}

// hostState is the fact at the top right of a host's sessions.
func (m *Model) hostState(h *hostConn) string {
	switch h.status {
	case stConnecting:
		return sKey.Render(m.spinner()) + sFaint.Render(" connecting")
	case stOffline:
		return sFaint.Render("offline")
	case stFailed:
		return sDanger.Render("not ready")
	}
	return sFaint.Render(fmtLatency(h.latency))
}

// Column widths of a session row.
const (
	colAgent = 8
	colState = 13
	colAge   = 4
)

// sessColumns is the width of the columns that share the free space.
type sessColumns struct {
	name, folder int
}

func sessCols(cw int) sessColumns {
	fixed := 3 + 2 + colAgent + 2 + 2 + colState + 2 + colAge + 1
	rest := max(cw-fixed, 12)
	c := sessColumns{name: min(max(rest*45/100, 10), 28)}
	c.folder = rest - c.name
	return c
}

func (m *Model) sessRowView(r sessRow, c sessColumns, selected bool) string {
	switch r.kind {
	case rowHeader:
		return " " + sBold.Render(r.h.name()) + "  " + m.hostState(r.h)
	case rowNote:
		msg, _ := errText(r.h.err)
		if msg == "" {
			msg = "not ready"
		}
		return "   " + sDanger.Render(glyphFail) + " " + sFaint.Render(trunc(sentence(msg), c.name+c.folder+20))
	}
	s := r.s
	glyph, label := m.look(s)
	name := seg{fit(s.Name, c.name), sBold}
	if m.rename != nil && m.rename.key == r.key() {
		name = seg{fit(m.rename.in.view(c.name, true), c.name), sPlain}
		selected = false
	}
	return listRow(selected,
		seg{" ", sPlain}, glyph, seg{" ", sPlain}, name, seg{"  ", sPlain},
		seg{fit(s.Agent, colAgent), sFaint}, seg{"  ", sPlain},
		seg{fit(truncPath(homePath(r.h, s.Cwd), c.folder), c.folder), sFaint}, seg{"  ", sPlain},
		seg{fit(label.s, colState), label.st}, seg{"  ", sPlain},
		seg{fitRight(fmtAge(m.now().Sub(since(s))), colAge), sFaint}, seg{" ", sPlain})
}

// emptyState explains an empty sessions list, and what to do next.
func (m *Model) emptyState(cw int) []string {
	if q := m.sessFilter.query(); q != "" {
		return []string{sFaint.Render(trunc(fmt.Sprintf("Nothing matches '%s'.", q), cw))}
	}
	h := m.selectedHost()
	if h == nil {
		loaded, connecting := false, false
		for _, x := range m.hosts {
			loaded = loaded || x.loaded
			connecting = connecting || x.status == stConnecting
		}
		var out []string
		switch {
		case loaded && m.hideExited:
			out = append(out, sFaint.Render("No running sessions."), "", keyHint(optKey+"+E", "show ended sessions"))
		case loaded:
			out = append(out, sFaint.Render("No sessions yet."), "", keyHint("Esc", "pick a host, then n for a new session"))
		case connecting:
			out = append(out, sKey.Render(m.spinner())+" "+sFaint.Render("Connecting…"))
		}
		for _, x := range m.hosts {
			if x.status == stOffline || x.status == stFailed {
				out = append(out, sFaint.Render(trunc(glyphOffline+" "+x.name()+" is offline.", cw)))
			}
		}
		return out
	}
	switch {
	case h.status == stOffline || h.status == stFailed:
		return m.hostError(h, cw)
	case !h.loaded:
		return []string{sKey.Render(m.spinner()) + " " + sFaint.Render(trunc("Connecting to "+h.name()+"…", cw-2))}
	case len(h.sessions) > 0 && m.hideExited:
		return []string{sFaint.Render("No running sessions."), "", keyHint(optKey+"+E", "show ended sessions")}
	default:
		return []string{sFaint.Render(trunc("No sessions on "+h.name()+" yet.", cw)), "", keyHint("n", "new session")}
	}
}

// keyHint is the faint hint of an empty list: "n new session".
func keyHint(key, desc string) string {
	return sKey.Faint(true).Render(key) + sFaint.Render(" "+desc)
}

// hostError says why a host cannot be used, and what to try, the way Amp
// prints an error: a bold red "Error:", the message, then the next step.
func (m *Model) hostError(h *hostConn, cw int) []string {
	msg, hint := errText(h.err)
	if msg == "" {
		msg = h.name() + " is offline"
	}
	if hint == "" {
		hint = fmt.Sprintf("%s host test %s", brand.Name, h.name())
	}
	return append(errorBlock(msg, hint, cw), "", sFaint.Render("Retrying in the background."))
}

// errorBlock is an error and the step to try next, wrapped to w cells.
func errorBlock(msg, hint string, w int) []string {
	var out []string
	for i, l := range wrap(sentence(msg), w-7) {
		lead := sErrHead.Render("Error:") + " "
		if i > 0 {
			lead = strings.Repeat(" ", 7)
		}
		out = append(out, lead+l)
	}
	if hint != "" {
		out = append(out, "")
		for _, l := range wrap(hint, w) {
			out = append(out, sLink.Render(l))
		}
	}
	return out
}
