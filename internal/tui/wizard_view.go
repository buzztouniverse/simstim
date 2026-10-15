package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/iAmLakshya/simstim/internal/brand"
)

func (w *wizard) view(m *Model) string {
	width := m.dialogWidth()
	cw := width - 4
	title := "Add a host"
	var body []string
	var hints []hint
	switch w.step {
	case wzWhere:
		body, hints = w.whereView(cw)
	case wzTest:
		body = append([]string{"", sBold.Render("Testing " + w.host.Name), ""}, w.checkLines(m, cw)...)
		hints = []hint{{"enter", "next"}, {"esc", "back"}}
		if w.err != nil {
			hints = []hint{{"enter", "try again"}, {"esc", "back"}}
		}
	case wzInstall:
		title = fmt.Sprintf("Install %s on %s?", brand.HostBinary, w.host.Name)
		body = w.installView(m, cw)
		hints = []hint{{"enter", "install"}, {"esc", "back"}}
	case wzReady:
		body = w.readyView(m, cw)
		hints = []hint{{"enter", "start a session"}, {"esc", "close"}}
		if !w.saved {
			hints = []hint{{"esc", "back"}}
			if w.err != nil {
				hints = []hint{{"enter", "try again"}, {"esc", "back"}}
			}
		}
	}
	if w.err != nil {
		body = append(append(body, ""), errorLines(w.err, cw)...)
	}
	body = append(body, "")
	step := sFaint.Render(fmt.Sprintf("step %d of %d", int(w.step)+1, wizSteps))
	return sheet(width, title, step, hints, trimBlank(body), toneNormal)
}

func (w *wizard) whereView(cw int) ([]string, []hint) {
	const lw = 13
	label := func(i int, s string) string {
		if w.field == i {
			return sBold.Render(fit(s, lw))
		}
		return sFaint.Render(fit(s, lw))
	}
	fw := cw - lw
	body := []string{
		"",
		sBold.Render("Where does it live?"),
		"",
		label(0, "SSH target") + w.in[0].view(fw, w.field == 0),
	}
	if list := w.matches(); len(list) > 0 {
		shown := strings.Join(list, ", ")
		count := fmt.Sprintf("  (%d found)", len(list))
		body = append(body, strings.Repeat(" ", lw)+sFaint.Render(trunc("From your SSH config: "+shown, fw-cells(count))+count))
	} else if len(w.known) > 0 {
		body = append(body, strings.Repeat(" ", lw)+sFaint.Render(trunc("Nothing in your SSH config matches.", fw)))
	}
	body = append(body,
		label(1, "Name")+w.in[1].view(fw, w.field == 1),
		label(2, "SSH config")+w.in[2].view(fw, w.field == 2),
	)
	hints := []hint{{"↑↓", "pick from SSH config"}, {"tab", "next field"}, {"enter", "test"}, {"esc", "cancel"}}
	if w.field != 0 {
		hints[0] = hint{"↑↓", "field"}
	}
	return body, hints
}

func (w *wizard) checkLines(m *Model, cw int) []string {
	var out []string
	for _, c := range w.checks {
		var mark string
		switch c.state {
		case ckRunning:
			mark = sKey.Render(m.spinner())
		case ckOK:
			mark = sSuccess.Render(glyphOK)
		case ckFailed:
			mark = sDanger.Render(glyphFail)
		default:
			mark = sFaint.Render(glyphIdle)
		}
		detail := c.detail
		if c.took >= time.Millisecond {
			detail = strings.TrimSpace(detail + " " + fmtLatency(c.took))
		}
		out = append(out, mark+" "+fit(c.name, 12)+sFaint.Render(trunc(detail, cw-14)))
	}
	return out
}

func (w *wizard) installView(m *Model, cw int) []string {
	ver := m.be.HelperVersion()
	size := ""
	if n, ok := m.be.HelperSize(w.plat); ok {
		size = fmt.Sprintf(", about %.0f MB", float64(n)/1e6)
	}
	field := func(k, v string) string { return "  " + sFaint.Render(fit(k, 8)) + trunc(v, cw-10) }
	body := []string{""}
	body = append(body, wrap(brand.DisplayName+" needs a small helper on the host to keep sessions alive.", cw)...)
	body = append(body, "",
		field("What", brand.HostBinary+" "+ver+size+", SHA-256 checked"),
		field("Where", fmt.Sprintf("~/.local/share/%s/bin/%s/", brand.Name, ver)),
		field("Also", "turn on linger, install this terminal's terminfo"),
		"")
	body = append(body, wrap("Nothing listens on the network, and nothing runs until you start a session.", cw)...)
	if w.run || len(w.checks) > 0 && w.checks[0].state == ckFailed {
		body = append(append(body, ""), w.checkLines(m, cw)...)
	}
	return body
}

func (w *wizard) readyView(m *Model, cw int) []string {
	body := append([]string{""}, w.checkLines(m, cw)...)
	if !w.saved || w.info == nil {
		return body
	}
	i := w.info
	var names []string
	for k := range i.Agents {
		names = append(names, k)
	}
	sort.Strings(names)
	field := func(k, v string) string { return "  " + sFaint.Render(fit(k, 8)) + trunc(v, cw-10) }
	sys := []string{}
	for _, s := range []string{i.Distro, i.Arch, plural(i.CPUs, "CPU", "CPUs"), fmtSize(i.MemTotal)} {
		if s != "" && !strings.HasPrefix(s, "0 ") {
			sys = append(sys, s)
		}
	}
	return append(body,
		"",
		sSuccess.Render(glyphOK)+" "+sBold.Render(w.host.Name+" is ready."),
		"",
		field("Agents", strings.Join(names, dot)),
		field("System", strings.Join(sys, dot)),
		field("Limits", fmtSize(i.MemoryMax)+" per session"),
	)
}
