package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// tone colours a frame and its title.
type tone int

const (
	toneNormal tone = iota
	toneDanger
)

func (t tone) styles() (line, title lipgloss.Style) {
	if t == toneDanger {
		return sDanger, sErrHead
	}
	return sPlain, sTitle
}

// box draws a notched frame exactly w cells wide and h rows high (docs/ux.md
// 2): a rounded border in the terminal's own colour, the title set into the
// top border, a fact at the top right, and key hints at the bottom right.
// Each body line gets one cell of padding on both sides, so it should be w-4
// cells wide; longer lines are cut.
func box(w, h int, title, right, bottom string, body []string, t tone) string {
	w, h = max(w, 8), max(h, 2)
	line, _ := t.styles()
	var b strings.Builder
	b.WriteString(topBorder(w, title, right, t))
	side := line.Render("│")
	for i := range h - 2 {
		l := ""
		if i < len(body) {
			l = body[i]
		}
		b.WriteString("\n" + side + " " + fit(l, w-4) + " " + side)
	}
	b.WriteString("\n" + bottomBorder(w, bottom, t))
	return b.String()
}

func topBorder(w int, title, right string, t tone) string {
	line, tst := t.styles()
	inner := w - 2
	rw := 0
	if cells(right) == 0 {
		right = ""
	}
	if right != "" {
		right = trunc(right, max(inner/2-4, 0))
		rw = cells(right) + 3 // " right ─"
	}
	if title != "" {
		title = trunc(title, max(inner-1-2-1-rw, 1))
	}
	out := line.Render("╭─")
	used := 1
	if title != "" {
		out += tst.Render(" " + title + " ")
		used += cells(title) + 2
	}
	out += line.Render(strings.Repeat("─", max(inner-used-rw, 0)))
	if right != "" {
		out += " " + right + " " + line.Render("─")
	}
	return out + line.Render("╮")
}

// bottomBorder carries hints or a fact, aligned to the right.
func bottomBorder(w int, bottom string, t tone) string {
	line, _ := t.styles()
	inner := w - 2
	if bottom == "" {
		return line.Render("╰" + strings.Repeat("─", inner) + "╯")
	}
	bottom = trunc(bottom, inner-4)
	fill := inner - 3 - cells(bottom)
	return line.Render("╰"+strings.Repeat("─", max(fill, 0))) + " " + bottom + " " + line.Render("─╯")
}

// hint is one key and what it does.
type hint struct{ key, desc string }

// hintLine draws hints the way a border shows them, "Enter attach · Esc
// back": keys in the key colour, words and dots faint. When they do not fit
// in w cells it drops them from the second-to-last backwards: the first
// hints are the most useful, and the last is the way out.
func hintLine(hs []hint, w int) string {
	hs = fitHints(hs, w)
	parts := make([]string, len(hs))
	for i, h := range hs {
		parts[i] = sKey.Render(h.key) + sFaint.Render(" "+h.desc)
	}
	return strings.Join(parts, sFaint.Render(dot))
}

// welcomeLine draws hints the way the welcome screen shows them, "ctrl+o for
// commands · ? for shortcuts": keys bold, words faint.
func welcomeLine(hs []hint, w int) string {
	hs = fitHints(hs, w)
	parts := make([]string, len(hs))
	for i, h := range hs {
		parts[i] = sBold.Render(h.key) + sFaint.Render(" "+h.desc)
	}
	return strings.Join(parts, sFaint.Render(dot))
}

func fitHints(hs []hint, w int) []hint {
	hs = append([]hint(nil), hs...)
	width := func(hs []hint) int {
		n := 0
		for i, h := range hs {
			if i > 0 {
				n += 3
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

// hintsFor fits hints into the bottom border of a box w cells wide.
func hintsFor(hs []hint, w int) string { return hintLine(hs, w-6) }

// listRow draws one row of a list. A selected row is a bar in the title
// colour with black text, bold kept: the one strong colour on the screen.
// Keys keep their own colour on the bar, as Amp's do.
func listRow(selected bool, segs ...seg) string {
	var b strings.Builder
	for _, sg := range segs {
		st := sg.st
		if selected {
			fg := colOnSel
			if sg.st.GetForeground() == colKey {
				fg = colKey
			}
			st = sPlain.Foreground(fg).Background(colTitle).Bold(sg.st.GetBold())
		}
		b.WriteString(st.Render(sg.s))
	}
	return b.String()
}

// layer is a block drawn over the screen at a position.
type layer struct {
	s    string
	x, y int
}

// compose draws the layers over base on a w×h canvas. With dim set, the
// backdrop is dimmed by setting the faint attribute on its cells, so it works
// with any theme, and backgrounds are dropped.
func compose(base string, w, h int, dim bool, layers ...layer) string {
	c := lipgloss.NewCanvas(w, h)
	uv.NewStyledString(base).Draw(c, c.Bounds())
	if dim {
		for y := range h {
			for x := range w {
				if cell := c.CellAt(x, y); cell != nil && (cell.Content != " " || !cell.Style.IsZero()) {
					cell.Style.Attrs |= uv.AttrFaint
					cell.Style.Attrs &^= uv.AttrBold
					if cell.Style.Bg != nil {
						// A selection bar goes quiet too: faint text, no bar.
						cell.Style.Bg, cell.Style.Fg = nil, nil
					}
				}
			}
		}
	}
	for _, l := range layers {
		lw, lh := lipgloss.Size(l.s)
		uv.NewStyledString(l.s).Draw(c, uv.Rect(l.x, l.y, lw, lh))
	}
	return c.Render()
}
