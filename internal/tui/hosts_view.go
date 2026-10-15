package tui

import (
	"fmt"
	"sort"
	"strings"
)

// Hosts.

func (m *Model) hostsView(cw int) listView {
	v := listView{
		crumb: sFaint.Render("hosts"),
		facts: m.needFacts(sFaint.Render(plural(len(m.hosts), "host", "hosts"))),
		hints: []hint{{"enter", "open"}, {"1–9", "jump"}, {"a", "add host"}, {"ctrl+o", "commands"}, {"?", "keys"}},
	}
	vh := m.visibleHosts()
	nameW := 8
	for _, h := range vh {
		nameW = max(nameW, cells(h.name()))
	}
	nameW = min(nameW, max(cw/4, 8))
	v.listed, v.top, v.sel = true, m.hostTop, m.hostSel
	for i := m.hostTop; i < len(vh) && i < m.hostTop+m.listRoom(); i++ {
		v.rows = append(v.rows, marker(i == m.hostSel)+m.hostRow(vh[i], i+1, i == m.hostSel, cw-2, nameW))
	}
	if h := m.selectedHost(); h != nil {
		v.detail = "  " + m.hostDetail(h, cw-2)
	}
	return v
}

// needFacts adds "1 needs you" to the facts, in yellow, when it is so.
func (m *Model) needFacts(facts string) string {
	if n := m.needYou(); n > 0 {
		return facts + sFaint.Render(dot) + sWarning.Render(fmt.Sprintf("%d needs you", n))
	}
	return facts
}

const hostRight = 6 // latency, or nothing

func (m *Model) hostRow(h *hostConn, n int, selected bool, cw, nameW int) string {
	detailW := cw - 4 - nameW - 2 - 2 - hostRight - 2 - traceCells
	glyph := seg{glyphOnline, sSuccess}
	nameSt := sBold
	var detail []seg
	right := seg{fitRight(fmtLatency(h.latency), hostRight), sFaint}
	blank := seg{strings.Repeat(" ", hostRight), sPlain}
	switch h.status {
	case stConnecting:
		glyph, right = seg{m.spinner(), sAccent}, blank
		text := "connecting…"
		if len(h.sessions) > 0 {
			text = m.summaryText(h) + dot + "connecting…"
		}
		detail = []seg{{text, sFaint}}
		if !selected {
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
	name := nameSegs(h.name(), nameW, nameSt, "")
	segs := make([]seg, 0, 10+len(name)+len(detail))
	segs = append(segs, gutter(n), seg{" ", sPlain}, glyph, seg{" ", sPlain})
	segs = append(segs, name...)
	segs = append(segs, seg{"  ", sPlain})
	segs = append(segs, detail...)
	segs = append(segs, seg{strings.Repeat(" ", max(detailW-used, 0)), sPlain}, seg{"  ", sPlain}, right, seg{"  ", sPlain},
		seg{m.hostTrace(h), sPlain})
	return listRow(selected, segs...)
}

// hostTrace is the trace of all of a host's sessions together.
func (m *Model) hostTrace(h *hostConn) string {
	k := sessKey{h.name(), ""}
	all := make([]timeList, 0, len(h.sessions))
	for _, s := range h.sessions {
		all = append(all, m.traces[sessKey{h.name(), s.ID}])
	}
	m.traces[k] = mergeTimes(all)
	st := sFaint
	if h.online() && len(m.traces[k]) > 0 {
		st = sAccent
	}
	return m.trace(k, st)
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
		names := make([]string, 0, len(i.Agents))
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
