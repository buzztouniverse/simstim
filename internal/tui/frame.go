package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// boxStyle colours a frame: its lines and its title.
type boxStyle struct {
	line  lipgloss.Style
	title lipgloss.Style
}

var (
	boxNormal  = boxStyle{line: sBorder, title: sBold}
	boxDanger  = boxStyle{line: sDanger, title: sDanger.Bold(true)}
	boxPreview = boxStyle{line: sBorder.Faint(true), title: sBold}
)

// box draws a rounded frame exactly w cells wide and h rows high: the title
// and its context in the top border, key hints or facts in the bottom border
// (docs/ux.md 1.2, "status lives in borders"). right and bottom may be styled.
func box(w, h int, title, right, bottom string, body []string, bs boxStyle) string {
	w, h = max(w, 6), max(h, 2)
	var b strings.Builder
	b.WriteString(topBorder(w, title, right, bs))
	side := bs.line.Render("│")
	for i := range h - 2 {
		l := ""
		if i < len(body) {
			l = body[i]
		}
		b.WriteString("\n" + side + fit(l, w-2) + side)
	}
	b.WriteString("\n" + bottomBorder(w, bottom, bs))
	return b.String()
}

func topBorder(w int, title, right string, bs boxStyle) string {
	inner := w - 2
	rw := 0
	if right != "" {
		rw = cells(right) + 3 // " right ─"
	}
	if title != "" {
		room := inner - 1 - 2 - 1 - rw // "─", spaces round the title, at least one "─"
		if room < 4 && rw > 0 {
			right, rw = "", 0
			room = inner - 4
		}
		title = trunc(title, room)
	}
	out := bs.line.Render("╭─")
	used := 1
	if title != "" {
		out += " " + bs.title.Render(title) + " "
		used += cells(title) + 2
	}
	out += bs.line.Render(strings.Repeat("─", max(inner-used-rw, 0)))
	if right != "" {
		out += " " + right + " " + bs.line.Render("─")
	}
	return out + bs.line.Render("╮")
}

func bottomBorder(w int, bottom string, bs boxStyle) string {
	inner := w - 2
	if bottom == "" {
		return bs.line.Render("╰" + strings.Repeat("─", inner) + "╯")
	}
	bottom = trunc(bottom, inner-4)
	fill := inner - 3 - cells(bottom)
	return bs.line.Render("╰─") + " " + bottom + " " + bs.line.Render(strings.Repeat("─", max(fill, 0))+"╯")
}

// hint is one key and what it does, for a bottom border.
type hint struct{ key, desc string }

// hintLine draws hints in at most w cells. When they do not fit, it drops
// them from the second-to-last backwards: the first hints are the most useful
// and the last one is always "? help" or "esc".
func hintLine(hs []hint, w int) string {
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
	for len(hs) > 2 && width(hs) > w {
		hs = append(hs[:len(hs)-2], hs[len(hs)-1])
	}
	parts := make([]string, len(hs))
	for i, h := range hs {
		parts[i] = sKey.Render(h.key) + " " + sMuted.Render(h.desc)
	}
	return strings.Join(parts, sMuted.Render(dot))
}

// hintsFor fits hints into the bottom border of a box w cells wide.
func hintsFor(hs []hint, w int) string { return hintLine(hs, w-6) }

// layer is a block drawn over the screen at a position.
type layer struct {
	s    string
	x, y int
}

// compose draws the layers over base on a w×h canvas. With dim set, the
// backdrop is dimmed by setting the dim attribute on its cells, so it works
// with any theme (docs/ux.md 2.5).
func compose(base string, w, h int, dim bool, layers ...layer) string {
	c := lipgloss.NewCanvas(w, h)
	uv.NewStyledString(base).Draw(c, c.Bounds())
	if dim {
		for y := range h {
			for x := range w {
				if cell := c.CellAt(x, y); cell != nil && (cell.Content != " " || !cell.Style.IsZero()) {
					cell.Style.Attrs |= uv.AttrFaint
					cell.Style.Attrs &^= uv.AttrBold
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
