package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
)

// The launcher (docs/ux.md 2.2 and 2.3) is one framed list in one column:
// the hosts, or the sessions on one host. Its title sits in the top border,
// its keys in the bottom border, one line about the selected row sits under
// the list, and short notices appear under the box.

// noticeRows are kept free under the box for notices, so the box never
// moves when one appears.
const noticeRows = 4

// boxGeo is where the launcher's box was last drawn: for the motion effects
// and for the mouse.
type boxGeo struct {
	x, y, w, h   int
	listY, listN int // the first list row on screen, and how many are drawn
	top          int // the list index of the first row drawn
}

// boxW is the width of the launcher's box.
func (m *Model) boxW() int { return max(min(m.width-4, 96), min(m.width, 32)) }

// boxTop is the row the box starts on: a little way down, the same on
// every screen, so moving between them does not shift it.
func (m *Model) boxTop() int { return min(max(m.height/6, 1), 5) }

// cw is the width of a row inside the box.
func (m *Model) cw() int { return m.boxW() - 4 }

// listRoom is how many rows the list may take.
func (m *Model) listRoom() int {
	room := m.height - m.boxTop() - noticeRows - 6 // borders, the blank rows, the detail line
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

// listView is what one screen puts in the box.
type listView struct {
	title, right string
	rows         []string // the list, or an empty state
	listed       bool     // rows are list rows, not an empty state
	top          int
	detail       string
	hints        []hint
}

// viewLauncher draws the box. It records where, in m.geo.
func (m *Model) viewLauncher() string {
	w, cw := m.boxW(), m.cw()
	var v listView
	if m.screen == scrHosts {
		v = m.hostsView(cw)
	} else {
		v = m.sessionsView(cw)
	}
	switch {
	case m.rename != nil:
		v.hints = []hint{{"Enter", "rename"}, {"Esc", "cancel"}}
	case m.activeFilter().typing:
		v.hints = []hint{{"Enter", "keep filter"}, {"↑↓", "move"}, {"Esc", "clear"}}
	}
	all := []string{""}
	if f := m.activeFilter(); f.shown() {
		all = append(all, "> "+f.in.view(cw-2, f.typing), "")
	}
	listAt := len(all)
	all = append(all, v.rows...)
	if v.detail != "" {
		all = append(all, "", v.detail)
	}
	if m.showHelp {
		all = append(all, "", strings.Repeat("─", cw))
		all = append(all, m.helpLines(cw)...)
	}
	all = append(all, "")
	b := strings.Split(box(w, len(all)+2, v.title, v.right, hintsFor(v.hints, w), all, toneNormal), "\n")

	x, top := max((m.width-w)/2, 0), m.boxTop()
	m.geo = boxGeo{x: x, y: top, w: w, h: len(b), listY: top + 1 + listAt, top: v.top}
	if v.listed {
		m.geo.listN = len(v.rows)
	}
	lines := make([]string, m.height)
	for i, l := range b {
		if top+i < m.height {
			lines[top+i] = strings.Repeat(" ", x) + l
		}
	}
	return strings.Join(lines, "\n")
}

// gutter is the row's number, faint, for the first nine rows: 1–9 open them.
func gutter(n int) seg {
	if n >= 1 && n <= 9 {
		return seg{fmt.Sprint(n), sFaint}
	}
	return seg{" ", sPlain}
}

// nameSegs is a name w cells wide, with the letters that match the filter
// in the key colour.
func nameSegs(name string, w int, st lipgloss.Style, q string) []seg {
	name = trunc(name, w)
	pad := seg{strings.Repeat(" ", max(w-cells(name), 0)), sPlain}
	hit := fuzzyPos(q, name)
	if len(hit) == 0 {
		return []seg{{name, st}, pad}
	}
	var out []seg
	for i, r := range []rune(name) {
		s := st
		if hit[i] {
			s = sKeyBold
		}
		if n := len(out); n > 0 && out[n-1].st.GetForeground() == s.GetForeground() {
			out[n-1].s += string(r)
			continue
		}
		out = append(out, seg{string(r), s})
	}
	return append(out, pad)
}

// Hosts.

func (m *Model) hostsView(cw int) listView {
	v := listView{
		title: "Hosts", right: sFaint.Render(plural(len(m.hosts), "host", "hosts")),
		hints: []hint{{"Enter", "open"}, {"n", "new session"}, {"a", "add host"}, {"Ctrl+O", "commands"}, {"?", "shortcuts"}},
	}
	vh := m.visibleHosts()
	if len(vh) == 0 {
		v.rows = []string{sFaint.Render(trunc(fmt.Sprintf("Nothing matches '%s'.", m.hostFilter.query()), cw))}
		return v
	}
	nameW := 8
	for _, h := range vh {
		nameW = max(nameW, cells(h.name()))
	}
	nameW = min(nameW, max(cw/3, 8))
	v.listed, v.top = true, m.hostTop
	for i := m.hostTop; i < len(vh) && i < m.hostTop+m.listRoom(); i++ {
		v.rows = append(v.rows, m.hostRow(vh[i], i+1, i == m.hostSel, cw, nameW))
	}
	if h := m.selectedHost(); h != nil {
		v.detail = m.hostDetail(h, cw)
	}
	return v
}

const hostRight = 8 // latency, or nothing

func (m *Model) hostRow(h *hostConn, n int, selected bool, cw, nameW int) string {
	detailW := cw - 4 - nameW - 2 - 2 - hostRight - 1
	glyph := seg{glyphOnline, sSuccess}
	nameSt := sBold
	var detail []seg
	right := seg{fitRight(fmtLatency(h.latency), hostRight), sFaint}
	blank := seg{strings.Repeat(" ", hostRight), sPlain}
	switch h.status {
	case stConnecting:
		glyph, right = seg{m.spinner(), sKey}, blank
		text := "connecting…"
		if len(h.sessions) > 0 {
			text = m.summaryText(h) + " · connecting…"
		}
		if selected {
			detail = []seg{{text, sFaint}}
		} else {
			detail = []seg{{m.shimmer(text), sPlain}}
		}
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
	segs := []seg{gutter(n), {" ", sPlain}, glyph, {" ", sPlain}}
	segs = append(segs, nameSegs(h.name(), nameW, nameSt, m.hostFilter.query())...)
	segs = append(segs, seg{"  ", sPlain})
	segs = append(segs, detail...)
	segs = append(segs, seg{strings.Repeat(" ", max(detailW-used, 0)), sPlain}, seg{"  ", sPlain}, right, seg{" ", sPlain})
	return listRow(selected, segs...)
}

// summaryText is a host's sessions in a few words: "4 sessions".
func (m *Model) summaryText(h *hostConn) string {
	if len(h.sessions) == 0 {
		return "no sessions"
	}
	return plural(len(h.sessions), "session", "sessions")
}

// hostSummary is the faint line about a host's sessions; the count that
// needs the user stands out.
func (m *Model) hostSummary(h *hostConn) []seg {
	if !h.loaded && len(h.sessions) == 0 {
		return []seg{{"…", sFaint}}
	}
	out := []seg{{m.summaryText(h), sFaint}}
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

// hostDetail is the line under the hosts: how it is reached, and what it is.
func (m *Model) hostDetail(h *hostConn, cw int) string {
	parts := []string{"ssh " + h.host.SSH}
	if i := h.info; i != nil {
		for _, s := range []string{i.Distro, plural(i.CPUs, "CPU", "CPUs")} {
			if s != "" && !strings.HasPrefix(s, "0 ") {
				parts = append(parts, s)
			}
		}
		var names []string
		for k := range i.Agents {
			names = append(names, k)
		}
		sort.Strings(names)
		if len(names) > 0 {
			parts = append(parts, strings.Join(names, ", "))
		}
	}
	return sFaint.Render(trunc(strings.Join(parts, dot), cw))
}

// Sessions.

func (m *Model) sessionsView(cw int) listView {
	v := listView{hints: []hint{{"Enter", "attach"}, {"n", "new"}, {"Space", "preview"}, {"Esc", "hosts"}, {"?", "shortcuts"}}}
	h := m.selectedHost()
	if h == nil {
		v.title, v.right = "All sessions", sFaint.Render(plural(len(m.hosts), "host", "hosts"))
	} else {
		v.title, v.right = h.name(), m.hostState(h)
	}
	rows := m.rows()
	if sessionRows(rows) == 0 {
		v.rows = m.emptyState(cw)
		return v
	}
	cols := sessCols(cw)
	n := 0
	for i := range m.sessTop {
		if rows[i].kind == rowSession {
			n++
		}
	}
	v.listed, v.top = true, m.sessTop
	for i := m.sessTop; i < len(rows) && i < m.sessTop+m.listRoom(); i++ {
		if rows[i].kind == rowSession {
			n++
		}
		v.rows = append(v.rows, m.sessRowView(rows[i], n, cols, i == m.selIdx))
	}
	if hc, s := m.selectedPair(); s != nil {
		v.detail = m.sessionDetail(hc, *s, cw)
	}
	return v
}

// hostState is the fact at the top right of a host's sessions.
func (m *Model) hostState(h *hostConn) string {
	switch h.status {
	case stConnecting:
		return sKey.Render(m.spinner()) + " " + m.shimmer("connecting")
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
	fixed := 4 + 2 + colAgent + 2 + 2 + colState + 2 + colAge + 1
	rest := max(cw-fixed, 12)
	c := sessColumns{name: min(max(rest*45/100, 10), 28)}
	c.folder = rest - c.name
	return c
}

func (m *Model) sessRowView(r sessRow, n int, c sessColumns, selected bool) string {
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
	if !r.h.loaded {
		// Remembered from the last run: shown at once, quiet until the host
		// answers.
		glyph.st, label.st = sFaint, sFaint
	}
	name := nameSegs(s.Name, c.name, sBold, m.sessFilter.query())
	if m.rename != nil && m.rename.key == r.key() {
		name = []seg{{fit(m.rename.in.view(c.name, true), c.name), sPlain}}
		selected = false
	}
	segs := []seg{gutter(n), {" ", sPlain}, glyph, {" ", sPlain}}
	segs = append(segs, name...)
	segs = append(segs,
		seg{"  ", sPlain},
		seg{fit(s.Agent, colAgent), sFaint}, seg{"  ", sPlain},
		seg{fit(truncPath(homePath(r.h, s.Cwd), c.folder), c.folder), sFaint}, seg{"  ", sPlain},
		seg{fit(label.s, colState), label.st}, seg{"  ", sPlain},
		seg{fitRight(fmtAge(m.now().Sub(since(s))), colAge), sFaint}, seg{" ", sPlain})
	return listRow(selected, segs...)
}

// sessionDetail is the line under the sessions: what the selected one is
// waiting for, where it runs, and its memory.
func (m *Model) sessionDetail(h *hostConn, s client.Session, cw int) string {
	var parts []string
	if s.State == "needs_input" && s.Reason != "" {
		parts = append(parts, sWarning.Render(trunc(s.Reason, cw/2)))
	}
	for _, p := range []string{homePath(h, s.Cwd), fmtMemory(s.MemoryCurrent, s.MemoryMax), s.ID} {
		if p != "" {
			parts = append(parts, sFaint.Render(p))
		}
	}
	if !h.loaded {
		parts = append(parts, sFaint.Render("last seen before this run"))
	}
	return trunc(strings.Join(parts, sFaint.Render(dot)), cw)
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
			out = append(out, sKey.Render(m.spinner())+" "+m.shimmer("Connecting…"))
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
		return []string{sKey.Render(m.spinner()) + " " + m.shimmer(trunc("Connecting to "+h.name()+"…", cw-2))}
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
