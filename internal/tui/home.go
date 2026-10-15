package tui

import (
	"fmt"
	"strings"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// homeLayout is the geometry of the home screen.
type homeLayout struct {
	bodyH  int // rows inside the outer frame
	hostsW int // the hosts pane, without the divider
	sessW  int // the sessions pane
	listH  int // rows for the session list
	prevH  int // rows for the preview box, borders included; 0 when hidden
}

func (m *Model) layout() homeLayout {
	inner := max(m.width-2, 20)
	l := homeLayout{bodyH: max(m.height-2, 2)}
	l.hostsW = min(max(inner*3/10, 22), 34)
	if inner-l.hostsW-1 < 36 {
		l.hostsW = max(inner/3, 12)
	}
	l.sessW = inner - l.hostsW - 1
	avail := l.bodyH - 1
	l.listH = avail
	if m.showPreview && avail >= 10 && m.sel != (sessKey{}) {
		l.listH = min(max(len(m.rows()), 1), avail-8)
		l.prevH = avail - l.listH - 1
	}
	return l
}

// viewHome draws hosts on the left, sessions on the right, the preview below
// them, and the status in the borders.
func (m *Model) viewHome() string {
	l := m.layout()
	left := m.hostsPane(l)
	right := m.sessionsPane(l)
	div := sBorder.Render("│")
	body := make([]string, l.bodyH)
	for i := range body {
		body[i] = fit(left[i], l.hostsW) + div + fit(right[i], l.sessW)
	}
	total, online := 0, 0
	for _, h := range m.hosts {
		total += len(h.sessions)
		if h.online() {
			online++
		}
	}
	ctx := plural(len(m.hosts), "host", "hosts") + dot + plural(total, "session", "sessions")
	return box(m.width, m.height, brand.DisplayName, sMuted.Render(ctx), hintsFor(m.homeHints(), m.width), body, boxNormal)
}

func (m *Model) homeHints() []hint {
	switch {
	case m.rename != nil:
		return []hint{{"enter", "rename"}, {"esc", "cancel"}}
	case m.typingFilter() != nil:
		return []hint{{"enter", "keep filter"}, {"↑↓", "move"}, {"esc", "clear"}}
	case m.focus == paneHosts:
		enter := hint{"enter", "new session"}
		if m.selectedHost() == nil {
			enter = hint{"enter", "sessions"}
		}
		return []hint{enter, {"tab", "sessions"}, {"a", "add host"}, {"ctrl+o", "commands"}, {"?", "help"}}
	default:
		return []hint{{"enter", "attach"}, {"n", "new"}, {"d", "delete"}, {"r", "rename"}, {"/", "filter"}, {"ctrl+o", "commands"}, {"?", "help"}}
	}
}

// Hosts pane.

const hostsRight = 9 // count and latency

func (m *Model) hostsPane(l homeLayout) []string {
	out := make([]string, 0, l.bodyH)
	hdr := " " + sLabel.Render("HOSTS")
	if f := &m.hostFilter; f.typing || f.query() != "" {
		hdr += " " + sPrimary.Render("/") + f.in.view(l.hostsW-9, f.typing)
	}
	out = append(out, hdr)
	vh := m.visibleHosts()
	for i := m.hostTop; i <= len(vh) && len(out) < l.bodyH; i++ {
		out = append(out, m.hostRow(i, vh, l.hostsW))
	}
	if len(vh) == 0 && len(out) < l.bodyH {
		out = append(out, " "+sMuted.Render(trunc(fmt.Sprintf("Nothing matches '%s'.", m.hostFilter.query()), l.hostsW-2)))
	}
	for len(out) < l.bodyH {
		out = append(out, "")
	}
	return out
}

func (m *Model) hostRow(i int, vh []*hostConn, w int) string {
	nameW := w - 1 - 2 - hostsRight - 1
	selected := i == m.hostSel
	reverse := selected && m.focus == paneHosts
	marker := seg{" ", sPlain}
	if selected {
		marker = seg{glyphPointer, sPrimary}
	}
	nameSt := sPlain
	if selected {
		nameSt = sBold
	}
	var right []seg
	var name string
	if i == 0 {
		name = "all hosts"
		total := 0
		for _, h := range m.hosts {
			total += len(h.sessions)
		}
		right = []seg{{fitRight(fmt.Sprint(total), 3), sPlain}, {strings.Repeat(" ", 6), sPlain}}
	} else {
		h := vh[i-1]
		name = h.name()
		count := ""
		if h.loaded {
			count = fmt.Sprint(len(h.sessions))
		}
		switch h.status {
		case stOnline:
			right = []seg{{fitRight(count, 3), sPlain}, {fitRight(fmtLatency(h.latency), 6), sMuted}}
		case stConnecting:
			right = []seg{{fitRight(count, 3), sPlain}, {fitRight(m.spinner(), 6), sPrimary}}
		case stOffline:
			right = []seg{{fitRight("offline", hostsRight), sMuted}}
			nameSt = nameSt.Foreground(colMuted)
			if !selected {
				marker = seg{glyphOffline, sMuted}
			}
		case stFailed:
			right = []seg{{fitRight("error", hostsRight), sDanger}}
			if !selected {
				marker = seg{glyphFail, sDanger}
			}
		}
	}
	segs := []seg{{" ", sPlain}, marker, {" ", sPlain}, {fit(name, nameW), nameSt}}
	segs = append(segs, right...)
	segs = append(segs, seg{" ", sPlain})
	return row(reverse, segs...)
}

// Sessions pane.

// Column widths of a session row.
const (
	colAgent = 8
	colState = 13
	colAge   = 4
)

func (m *Model) sessionsPane(l homeLayout) []string {
	w := l.sessW
	out := make([]string, 0, l.bodyH)
	out = append(out, m.sessHeader(w))
	rows := m.rows()
	if sessionRows(rows) == 0 {
		out = append(out, m.emptyState(w)...)
	} else {
		cols := sessCols(w)
		for i := m.sessTop; i < len(rows) && i < m.sessTop+l.listH; i++ {
			out = append(out, m.sessRowView(rows[i], cols, i == m.selIdx))
		}
		for len(out) < 1+l.listH {
			out = append(out, "")
		}
		if l.prevH > 0 {
			out = append(out, "")
			for _, ln := range strings.Split(m.previewBox(w-2, l.prevH), "\n") {
				out = append(out, " "+ln)
			}
		}
	}
	if len(out) > l.bodyH {
		out = out[:l.bodyH]
	}
	for len(out) < l.bodyH {
		out = append(out, "")
	}
	return out
}

func (m *Model) sessHeader(w int) string {
	where := "all hosts"
	if h := m.selectedHost(); h != nil {
		where = h.name()
	}
	s := " " + sLabel.Render("SESSIONS") + sBorder.Render(" ─ ") + sSecondary.Render(where)
	if m.hideExited {
		s += sMuted.Render(dot + "exited hidden")
	}
	if f := &m.sessFilter; f.typing || f.query() != "" {
		room := w - cells(s) - 3
		s += "  " + sPrimary.Render("/") + f.in.view(max(room, 4), f.typing)
	}
	return s
}

// sessColumns is the width of each column of a session row.
type sessColumns struct {
	name, folder int
}

func sessCols(w int) sessColumns {
	fixed := 1 + 1 + 1 + 1 + colAgent + 1 + colState + 1 + colAge + 1
	c := sessColumns{name: max(w-fixed, 6)}
	if c.name >= 40 {
		c.folder = min(30, c.name-22)
		c.name -= c.folder + 1
	}
	return c
}

func (m *Model) sessRowView(r sessRow, c sessColumns, selected bool) string {
	switch r.kind {
	case rowHeader:
		name := r.h.name()
		return " " + sSecondary.Render(name) + " " + sBorder.Render(strings.Repeat("─", 3))
	case rowNote:
		msg, _ := errText(r.h.err)
		note := "offline"
		if r.h.status == stFailed && msg != "" {
			note = msg
		}
		return "   " + sMuted.Render(glyphOffline+" "+note)
	}
	s := r.s
	glyph, gst, label := m.look(s)
	reverse := selected && m.focus == paneSessions
	nameSt := sPlain
	if selected {
		nameSt = sBold
	}
	name := seg{fit(s.Name, c.name), nameSt}
	if m.rename != nil && m.rename.key == r.key() {
		name = seg{fit(m.rename.in.view(c.name, true), c.name), sPlain}
		reverse = false
	}
	segs := []seg{{" ", sPlain}, {glyph, gst.st}, {" ", sPlain}, name, {" ", sPlain}, {fit(s.Agent, colAgent), sPlain}, {" ", sPlain}}
	if c.folder > 0 {
		segs = append(segs, seg{fit(truncPath(homePath(r.h, s.Cwd), c.folder), c.folder), sSecondary}, seg{" ", sPlain})
	}
	segs = append(segs,
		seg{fit(label.s, colState), label.st}, seg{" ", sPlain},
		seg{fitRight(fmtAge(m.now().Sub(since(s))), colAge), sMuted}, seg{" ", sPlain})
	return row(reverse, segs...)
}

// emptyState explains an empty sessions pane, and what to do next.
func (m *Model) emptyState(w int) []string {
	pad := func(s string) string { return " " + s }
	if q := m.sessFilter.query(); q != "" {
		return []string{pad(sMuted.Render(trunc(fmt.Sprintf("Nothing matches '%s'.", q), w-2)))}
	}
	h := m.selectedHost()
	if h == nil {
		var out []string
		loaded, connecting := false, false
		for _, x := range m.hosts {
			loaded = loaded || x.loaded
			connecting = connecting || x.status == stConnecting
		}
		switch {
		case loaded && m.hideExited:
			out = append(out, pad(sMuted.Render("No running sessions. Press alt+e to show ended ones.")))
		case loaded:
			out = append(out, pad(sMuted.Render("No sessions yet. Choose a host and press enter.")))
		case connecting:
			out = append(out, pad(sPrimary.Render(m.spinner())+" "+sMuted.Render("Connecting…")))
		}
		for _, x := range m.hosts {
			if x.status == stOffline || x.status == stFailed {
				out = append(out, pad(sMuted.Render(trunc(glyphOffline+" "+x.name()+" is offline.", w-2))))
			}
		}
		return out
	}
	switch {
	case h.status == stOffline || h.status == stFailed:
		return m.hostError(h, w)
	case !h.loaded:
		return []string{pad(sPrimary.Render(m.spinner()) + " " + sMuted.Render(trunc("Connecting to "+h.name()+"…", w-4)))}
	case len(h.sessions) > 0 && m.hideExited:
		return []string{pad(sMuted.Render(trunc(fmt.Sprintf("No running sessions on %s. Press alt+e to show ended ones.", h.name()), w-2)))}
	default:
		return wrapPad(fmt.Sprintf("No sessions on %s. Press enter to start one.", h.name()), w, sMuted)
	}
}

// hostError shows why a host cannot be used, and what to try.
func (m *Model) hostError(h *hostConn, w int) []string {
	msg, hint := errText(h.err)
	if hint == "" {
		hint = fmt.Sprintf("%s host test %s", brand.Name, h.name())
	}
	title := h.name() + " is offline."
	glyph := sMuted.Render(glyphOffline)
	if h.status == stFailed {
		title = h.name() + " is not ready."
		glyph = sDanger.Render(glyphFail)
	}
	out := []string{" " + glyph + " " + sBold.Render(trunc(title, w-4)), ""}
	if msg != "" {
		for _, l := range wrap(sentence(msg), w-4) {
			out = append(out, "   "+sMuted.Render(l))
		}
		out = append(out, "")
	}
	if hint != "" {
		for i, l := range wrap(hint, w-9) {
			lead := "   " + sPrimary.Render("Try:") + " "
			if i > 0 {
				lead = strings.Repeat(" ", 8)
			}
			out = append(out, lead+l)
		}
		out = append(out, "")
	}
	return append(out, "   "+sMuted.Render("Retrying in the background."))
}

func wrapPad(s string, w int, st interface{ Render(...string) string }) []string {
	var out []string
	for _, l := range wrap(s, w-2) {
		out = append(out, " "+st.Render(l))
	}
	return out
}

// previewBox frames the selected session's last screen. The bottom border
// shows the agent, the folder and the memory use.
func (m *Model) previewBox(w, h int) string {
	hc, s := m.selectedPair()
	if s == nil {
		return ""
	}
	var facts []string
	for _, f := range []string{s.Agent, truncPath(homePath(hc, s.Cwd), max(w/3, 16)), fmtMemory(s.MemoryCurrent, s.MemoryMax)} {
		if f != "" {
			facts = append(facts, f)
		}
	}
	bottom := sMuted.Render(strings.Join(facts, dot))
	body := m.previewLines(w-4, h-2)
	for i := range body {
		body[i] = " " + body[i]
	}
	return box(w, h, s.Name, "", bottom, body, boxPreview)
}
