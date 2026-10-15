package main

import (
	"fmt"
	"image/color"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

// The SVG is a picture of the terminal window, for the README. Every run of
// text is pinned to its column and stretched to its exact width, so the grid
// holds in whatever monospace font the viewer has, including the fallback
// font a browser picks for braille and other symbols.

const (
	fontSize = 14.0
	cellW    = 8.4 // 0.6em: the advance of most monospace fonts
	cellH    = 17.0
	baseline = 13.0 // from the top of a row
	padX     = 24.0
	padTop   = 44.0 // room for the window buttons
	padBot   = 22.0
)

// shotTheme is the palette the images use: the reference theme of the
// palette table in docs/ux.md 4.1, a dark Ghostty theme. The UI sends palette
// slots, so in a real terminal the user's own theme decides the colours.
// Faint text is drawn half way to the background, as Ghostty draws it.
var shotTheme = struct {
	fg, bg  string
	palette [16]string
}{
	fg: "#BFBDB6", bg: "#0B0E14",
	palette: [16]string{
		"#1E232B", "#EA6C73", "#7FD962", "#F9AF4F", "#53BDFA", "#CDA1FA", "#90E1C6", "#C7C7C7",
		"#686868", "#F07178", "#AAD94C", "#FFB454", "#59C2FF", "#D2A6FF", "#95E6CB", "#FFFFFF",
	},
}

// look is a cell's resolved appearance.
type look struct {
	fg, bg                       string
	bold, italic, under, through bool
}

func resolve(s uv.Style) look {
	fg, bg := hexOf(s.Fg, shotTheme.fg), hexOf(s.Bg, shotTheme.bg)
	if s.Attrs&uv.AttrReverse != 0 {
		fg, bg = bg, fg
	}
	if s.Attrs&uv.AttrFaint != 0 {
		fg = mix(fg, bg)
	}
	if s.Attrs&uv.AttrConceal != 0 {
		fg = bg
	}
	return look{fg: fg, bg: bg, bold: s.Attrs&uv.AttrBold != 0, italic: s.Attrs&uv.AttrItalic != 0,
		under: s.Underline != 0, through: s.Attrs&uv.AttrStrikethrough != 0}
}

func hexOf(c color.Color, def string) string {
	switch v := c.(type) {
	case nil:
		return def
	case ansi.BasicColor:
		return shotTheme.palette[v&15]
	case ansi.IndexedColor:
		if v < 16 {
			return shotTheme.palette[v]
		}
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

// mix is halfway between two colours: how terminals draw faint text.
func mix(a, b string) string {
	var ar, ag, ab, br, bg, bb int
	_, _ = fmt.Sscanf(a, "#%02X%02X%02X", &ar, &ag, &ab)
	_, _ = fmt.Sscanf(b, "#%02X%02X%02X", &br, &bg, &bb)
	return fmt.Sprintf("#%02X%02X%02X", (ar+br)/2, (ag+bg)/2, (ab+bb)/2)
}

type run struct {
	x, cells int
	text     strings.Builder
	look     look
}

// svgScreen draws the emulator's screen as a terminal window.
func svgScreen(emu *vt.SafeEmulator) string {
	w, h := emu.Width(), emu.Height()
	width, height := padX*2+float64(w)*cellW, padTop+padBot+float64(h)*cellH
	var bgs, texts strings.Builder
	for y := 0; y < h; y++ {
		top := padTop + float64(y)*cellH
		var runs []*run
		for x := 0; x < w; x++ {
			c := emu.CellAt(x, y)
			if c != nil && c.Width == 0 && c.Content == "" {
				continue // the second half of a wide character
			}
			content, cells, lk := " ", 1, resolve(uv.Style{})
			if c != nil {
				lk = resolve(c.Style)
				if c.Content != "" {
					content, cells = c.Content, max(c.Width, 1)
				}
			}
			// A symbol gets a run of its own, so a fallback font cannot shift
			// the text after it.
			symbol := len(content) > 1 && !isBoxDrawing(content)
			last := len(runs) - 1
			if last < 0 || runs[last].look != lk || symbol || runs[last].symbol() {
				runs = append(runs, &run{x: x, look: lk})
				last++
			}
			runs[last].text.WriteString(content)
			runs[last].cells += cells
		}
		// Backgrounds, with neighbours of one colour merged so no seam shows.
		for i := 0; i < len(runs); {
			j, cells := i, 0
			for ; j < len(runs) && runs[j].look.bg == runs[i].look.bg; j++ {
				cells += runs[j].cells
			}
			if runs[i].look.bg != shotTheme.bg {
				fmt.Fprintf(&bgs, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`+"\n",
					padX+float64(runs[i].x)*cellW, top, float64(cells)*cellW, cellH, runs[i].look.bg)
			}
			i = j
		}
		for _, r := range runs {
			x := padX + float64(r.x)*cellW
			span := float64(r.cells) * cellW
			text := r.text.String()
			if strings.TrimSpace(text) == "" && !r.look.under && !r.look.through {
				continue
			}
			fmt.Fprintf(&texts, `<text x="%.1f" y="%.1f" textLength="%.1f" fill="%s"%s>%s</text>`+"\n",
				x, top+baseline, span, r.look.fg, attrs(r.look), escapeXML(text))
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`+"\n",
		width, height, width, height)
	fmt.Fprintf(&b, `<style>text{font-family:"JetBrains Mono","SF Mono",Menlo,Consolas,"DejaVu Sans Mono",monospace;font-size:%.0fpx;white-space:pre}</style>`+"\n", fontSize)
	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" rx="10" fill="%s"/>`+"\n", width, height, shotTheme.bg)
	for i, c := range []string{"#FF5F57", "#FEBC2E", "#28C840"} {
		fmt.Fprintf(&b, `<circle cx="%d" cy="20" r="6" fill="%s"/>`+"\n", 22+i*20, c)
	}
	b.WriteString(bgs.String())
	b.WriteString(texts.String())
	b.WriteString("</svg>\n")
	return b.String()
}

func (r *run) symbol() bool {
	t := r.text.String()
	return len(t) > 1 && r.cells == 1 && !isBoxDrawing(t)
}

// isBoxDrawing is true for the line characters, which every monospace font
// draws at the cell width.
func isBoxDrawing(s string) bool {
	for _, r := range s {
		if r < 0x2500 || r > 0x257F {
			return false
		}
	}
	return true
}

func attrs(l look) string {
	var a []string
	if l.bold {
		a = append(a, `font-weight="bold"`)
	}
	if l.italic {
		a = append(a, `font-style="italic"`)
	}
	switch {
	case l.under && l.through:
		a = append(a, `text-decoration="underline line-through"`)
	case l.under:
		a = append(a, `text-decoration="underline"`)
	case l.through:
		a = append(a, `text-decoration="line-through"`)
	}
	if len(a) == 0 {
		return ""
	}
	return " " + strings.Join(a, " ")
}

func escapeXML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
