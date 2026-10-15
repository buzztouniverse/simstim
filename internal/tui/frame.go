package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// tone colours a sheet's rule and title.
type tone int

const (
	toneNormal tone = iota
	toneDanger
)

func (t tone) styles() (line, title lipgloss.Style) {
	if t == toneDanger {
		return sDanger, sErrHead
	}
	return sFaint, sTitle
}

// rule is a faint line w cells wide with a title set into it near the left
// and a fact near the right: "── Title ──────── fact ──".
func rule(w int, title, right string, t tone) string {
	line, tst := t.styles()
	if cells(right) == 0 {
		right = ""
	}
	out := line.Render("──")
	used := 2
	if title != "" {
		title = trunc(title, max(w-8-cells(right), 1))
		out += " " + tst.Render(title) + " "
		used += cells(title) + 2
	}
	rw := 0
	if right != "" {
		rw = cells(right) + 4 // " right ──"
	}
	out += line.Render(strings.Repeat("─", max(w-used-rw, 0)))
	if right != "" {
		out += " " + right + " " + line.Render("──")
	}
	return out
}

// sheet draws a dialog (docs/ux.md 2.4 to 2.12): it rises from the bottom
// of the screen, full width, with its title in a rule across the top, its
// body indented under it, and its keys on the last line. It has no side
// borders; the dimmed list behind it is the frame.
func sheet(w int, title, right string, hints []hint, body []string, t tone) string {
	lines := []string{rule(w, title, right, t), ""}
	for _, l := range body {
		lines = append(lines, "  "+fit(l, w-4))
	}
	lines = append(lines, "", "  "+hintLine(hints, w-4))
	for i, l := range lines {
		lines[i] = fit(l, w) // opaque, so the list behind does not show through
	}
	return strings.Join(lines, "\n")
}

// hint is one key and what it does.
type hint struct{ key, desc string }

// hintLine draws hints as the footer shows them, "enter attach   n new":
// keys in the accent colour, words faint, wide gaps instead of dots. When
// they do not fit in w cells it drops them from the second-to-last
// backwards: the first hints are the most useful, and the last is the way
// out.
func hintLine(hs []hint, w int) string {
	hs = fitHints(hs, w)
	parts := make([]string, len(hs))
	for i, h := range hs {
		parts[i] = sKey.Render(h.key) + sFaint.Render(" "+h.desc)
	}
	return strings.Join(parts, sep)
}

// welcomeLine draws hints the way the welcome screen shows them: keys bold,
// words faint.
func welcomeLine(hs []hint, w int) string {
	hs = fitHints(hs, w)
	parts := make([]string, len(hs))
	for i, h := range hs {
		parts[i] = sBold.Render(h.key) + sFaint.Render(" "+h.desc)
	}
	return strings.Join(parts, sep)
}

func fitHints(hs []hint, w int) []hint {
	hs = append([]hint(nil), hs...)
	width := func(hs []hint) int {
		n := 0
		for i, h := range hs {
			if i > 0 {
				n += len(sep)
			}
			n += cells(h.key) + 1 + cells(h.desc)
		}
		return n
	}
	for len(hs) > 1 && width(hs) > w {
		if len(hs) == 2 {
			hs = hs[1:]
			continue
		}
		hs = append(hs[:len(hs)-2], hs[len(hs)-1])
	}
	return hs
}

// listRow draws one row of a list. The selected row is lifted: its faint
// parts turn to normal, and the layout puts the marker in front of it.
func listRow(selected bool, segs ...seg) string {
	var b strings.Builder
	for _, sg := range segs {
		st := sg.st
		if selected && st.GetFaint() {
			st = st.Faint(false)
		}
		b.WriteString(st.Render(sg.s))
	}
	return b.String()
}

// marker is the gutter in front of a row: the accent bar when selected.
func marker(selected bool) string {
	if selected {
		return sAccent.Render(glyphMarker) + " "
	}
	return "  "
}

// dim quiets everything drawn so far, behind a sheet: the faint attribute on
// every cell, so it works with any theme, bold removed, and backgrounds
// dropped.
func dim(c *lipgloss.Canvas) {
	b := c.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if cell := c.CellAt(x, y); cell != nil && (cell.Content != " " || !cell.Style.IsZero()) {
				cell.Style.Attrs |= uv.AttrFaint
				cell.Style.Attrs &^= uv.AttrBold
				if cell.Style.Bg != nil {
					cell.Style.Bg, cell.Style.Fg = nil, nil
				}
			}
		}
	}
}

// trimBlank drops blank lines at the start and the end of a sheet's body:
// the sheet adds its own.
func trimBlank(body []string) []string {
	for len(body) > 0 && strings.TrimSpace(body[0]) == "" {
		body = body[1:]
	}
	for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
		body = body[:len(body)-1]
	}
	return body
}
