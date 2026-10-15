package tui

import (
	"fmt"
	"strings"

	"github.com/iAmLakshya/simstim/internal/onboard"
)

func (g *guide) view(m *Model) string {
	width := m.dialogWidth()
	cw := width - 4
	title, right := "Set up", ""
	var body, tail []string
	var hints []hint
	switch g.step {
	case gdWhere:
		body, hints = g.whereView(cw)
	case gdName, gdCreate:
		p, _ := onboard.FindPlace(g.places, g.kind)
		title = p.Title
		body, hints = g.createView(m, cw)
	default:
		title = "A server I will rent"
		n := min(int(g.step-gdRentKey), onboard.RentNameStep)
		right = sFaint.Render(fmt.Sprintf("step %d of %d", n+1, len(onboard.RentTitles)))
		body, tail, hints = g.rentView(m, cw)
	}
	if g.err != nil {
		tail = append(append(tail, ""), errorLines(g.err, cw)...)
	}
	return sheet(width, title, right, hints, fitBody(body, tail, m.height), toneNormal)
}

// fitBody joins a sheet's body and its tail (a field, progress, an error),
// and keeps them within a screen h rows tall, with room for the sheet's
// rule, its keys and the gaps between them. The body gives way first: the
// tail is what the user acts on.
func fitBody(body, tail []string, h int) []string {
	body, tail = trimBlank(body), trimBlank(tail)
	if len(tail) > 0 {
		tail = append([]string{""}, tail...)
	}
	room := max(h-5, 3)
	if len(tail) >= room {
		return tail[len(tail)-room:]
	}
	if len(body)+len(tail) > room {
		body = append(body[:room-len(tail)-1:room-len(tail)-1], sFaint.Render("…"))
	}
	return append(body, tail...)
}

func (g *guide) whereView(cw int) ([]string, []hint) {
	body := []string{"", sBold.Render("Where will agents run?"), ""}
	if len(g.places) == 0 {
		return append(body, sFaint.Render("Looking at this computer…")), []hint{{"esc", "close"}}
	}
	const titleW = 34
	for i, p := range g.places {
		glyph := sSuccess.Render(glyphOnline)
		if !p.Ready {
			glyph = sFaint.Render(glyphIdle)
		}
		row := marker(i == g.sel) + listRow(i == g.sel,
			seg{fmt.Sprintf("%d ", i+1), sFaint}, seg{glyph + " ", sPlain},
			seg{fit(p.Title, titleW), sBold}, seg{"  " + trunc(p.Note, max(cw-titleW-8, 0)), sFaint})
		body = append(body, row)
	}
	hints := []hint{{"enter", "choose"}, {"↑↓", "move"}, {"esc", "close"}}
	if g.sel < len(g.places) {
		if p := g.places[g.sel]; !p.Ready {
			body = append(body, "", sBold.Render("To make it ready:"))
			body = append(body, stepLines(p.Steps, cw)...)
			hints = []hint{{"enter", "check again"}, {"↑↓", "move"}, {"esc", "close"}}
		}
	}
	return body, hints
}

// stepLines draws things to do: the words, then each command in the
// accent colour, indented.
func stepLines(ss []onboard.Step, cw int) []string {
	var out []string
	for _, s := range ss {
		if s.Say != "" {
			out = append(out, wrap(s.Say, cw)...)
		}
		if s.Run != "" {
			out = append(out, "  "+sAccent.Render(trunc(s.Run, cw-2)))
		}
	}
	return out
}

func (g *guide) createView(m *Model, cw int) ([]string, []hint) {
	const lw = 13
	body := []string{""}
	body = append(body, wrap("It makes the machine, installs the helper on it, and connects. Making it counts as your yes to the helper.", cw)...)
	body = append(body, "", sBold.Render(fit("Name", lw))+g.in.view(cw-lw, g.step == gdName))
	if g.step == gdName {
		return body, []hint{{"enter", "make it"}, {"esc", "back"}}
	}
	body = append(append(body, ""), m.checkLines(g.checks, cw)...)
	if l := g.out.String(); l != "" && g.run {
		body = append(body, "  "+sFaint.Render(trunc(l, cw-2)))
	}
	if g.note != "" {
		body = append(body, sFaint.Render(g.note))
	}
	if g.run {
		return body, []hint{{"esc", "stop"}}
	}
	return body, []hint{{"enter", "try again"}, {"esc", "back"}}
}

func (g *guide) rentView(m *Model, cw int) (body, tail []string, hints []hint) {
	const lw = 13
	n := min(int(g.step-gdRentKey), onboard.RentNameStep)
	body = []string{"", sBold.Render(onboard.RentTitles[n]), ""}
	field := func(label string) string {
		return sBold.Render(fit(label, lw)) + g.in.view(cw-lw, true)
	}
	switch g.step {
	case gdRentKey:
		body = append(body, stepLines(g.plan.KeySteps(), cw)...)
		hints = []hint{{"enter", "next"}, {"esc", "back"}}
		if g.plan.Key == "" {
			hints = []hint{{"k", "make it now"}, {"enter", "check again"}, {"esc", "back"}}
		}
	case gdRentServer:
		body = append(body, stepLines(g.plan.ServerSteps(), cw)...)
		tail = []string{field("Address")}
		hints = []hint{{"enter", "next"}, {"esc", "back"}}
	case gdRentUser:
		p := g.plan
		if u := strings.TrimSpace(g.in.value()); onboard.ValidUser(u) {
			p.User = u
		}
		body = append(body, stepLines(p.UserSteps(), cw)...)
		tail = []string{field("User")}
		hints = []hint{{"enter", "check that ssh " + p.Login() + " works"}, {"esc", "back"}}
	case gdRentName:
		body = append(body, wrap("A name for the server, in ~/.ssh/config and here.", cw)...)
		tail = []string{field("Name")}
		hints = []hint{{"enter", "next"}, {"esc", "back"}}
	case gdRentConfig:
		body, hints = g.configView(body, cw)
	}
	if len(g.checks) > 0 {
		tail = append(append(tail, ""), m.checkLines(g.checks, cw)...)
	}
	if g.run {
		hints = []hint{{"esc", "back"}}
	}
	return body, tail, hints
}

func (g *guide) configView(body []string, cw int) ([]string, []hint) {
	if g.present {
		body = append(body, wrap("Your ~/.ssh/config names "+g.plan.Name+". This checks that the name reaches the server.", cw)...)
		if g.note != "" {
			body = append(body, "", sSuccess.Render(glyphOK)+" "+g.note)
		}
		return body, []hint{{"enter", "check"}, {"esc", "back"}}
	}
	body = append(body, wrap("These lines give the server its name in ~/.ssh/config:", cw)...)
	body = append(body, "")
	for _, l := range strings.Split(strings.TrimRight(g.plan.ConfigLines(), "\n"), "\n") {
		body = append(body, "  "+sAccent.Render(trunc(l, cw-2)))
	}
	body = append(body, "")
	if g.asked {
		body = append(body, wrap("Add them yourself, at the end of ~/.ssh/config, then check.", cw)...)
		return body, []hint{{"enter", "check"}, {"esc", "back"}}
	}
	body = append(body, wrap("Add them to the end of ~/.ssh/config?", cw)...)
	return body, []hint{{"y", "add them"}, {"n", "I will add them"}, {"esc", "back"}}
}
