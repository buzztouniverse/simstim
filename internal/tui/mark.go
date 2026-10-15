package tui

import (
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// The welcome mark (docs/ux.md 2.1): three strands of a signal, woven into
// a ribbon that pinches at both ends, drawn in braille so the curves are
// smooth, in the brand gradient. A pulse runs along it, and the strands are
// brighter where they cross. With motion on it flows at 30 fps; with motion
// off it is one still frame.

const (
	markW     = 30
	markH     = 7
	markPulse = 2600 * time.Millisecond // one pulse along the ribbon
)

type strand struct{ amp, wave, speed, phase float64 }

var strands = []strand{
	{amp: 1.0, wave: 1.0, speed: 1.2, phase: 0},
	{amp: 0.8, wave: 1.6, speed: -0.9, phase: 2.2},
	{amp: 0.55, wave: 2.4, speed: 1.7, phase: 4.4},
}

// brailleBit is the dot of a braille cell at column x (0–1) and row y (0–3).
var brailleBit = [4][2]uint{{0, 3}, {1, 4}, {2, 5}, {6, 7}}

// markLines draws the mark at the current moment.
func (m *Model) markLines() []string {
	t := 0.0
	if m.motion {
		t = m.since(m.epoch).Seconds()
	}
	dw, dh := markW*2, markH*4
	dots := make([][]bool, dh)
	for i := range dots {
		dots[i] = make([]bool, dw)
	}
	mid := float64(dh-1) / 2
	for _, s := range strands {
		prev := -1
		for x := range dw {
			u := (float64(x) + 0.5) / float64(dw)
			env := math.Pow(math.Sin(math.Pi*u), 1.2)
			y := int(math.Round(mid + s.amp*env*(float64(dh)/2-1)*math.Sin(2*math.Pi*u*s.wave+s.speed*t+s.phase)))
			y = min(max(y, 0), dh-1)
			dots[y][x] = true
			if prev >= 0 { // join steep steps, so the line never breaks
				for yy := min(prev, y) + 1; yy < max(prev, y); yy++ {
					dots[yy][x] = true
				}
			}
			prev = y
		}
	}
	cols := m.gradient(markW)
	smooth := m.smooth()
	back := bg(m)
	pulse := -5 + math.Mod(t/markPulse.Seconds(), 1)*(markW+10)
	out := make([]string, markH)
	for cy := range markH {
		var b strings.Builder
		for cx := range markW {
			r, n := 0, 0
			for dy := range 4 {
				for dx := range 2 {
					if dots[cy*4+dy][cx*2+dx] {
						r |= 1 << brailleBit[dy][dx]
						n++
					}
				}
			}
			if r == 0 {
				b.WriteByte(' ')
				continue
			}
			glyph := string(rune(0x2800 + r))
			st := sBrand
			switch {
			case cols != nil && smooth:
				p := (float64(cx) - pulse) / 4
				k := clamp01(0.4 + 0.1*float64(n) + 0.55*math.Exp(-p*p))
				st = lipgloss.NewStyle().Foreground(mix(back, toRGB(cols[cx]), k).color())
			case cols != nil:
				st = lipgloss.NewStyle().Foreground(cols[cx])
			}
			b.WriteString(st.Render(glyph))
		}
		out[cy] = b.String()
	}
	return out
}

// welcomeTitle is "Welcome to <name>" in the brand gradient, bold. With
// motion on it types itself in, behind a cursor.
func (m *Model) welcomeTitle() string {
	text := []rune("Welcome to " + brand.DisplayName)
	shown := len(text)
	if m.motion {
		shown = min(int(m.since(m.epoch)/typeEvery), len(text))
	}
	cols := m.gradient(len(text))
	var b strings.Builder
	for i, r := range text[:shown] {
		st := sBold
		switch {
		case cols != nil:
			st = st.Foreground(cols[i])
		case i >= len(text)-len([]rune(brand.DisplayName)):
			st = sBrand.Bold(true)
		}
		b.WriteString(st.Render(string(r)))
	}
	if shown < len(text) {
		b.WriteString(sBrand.Render("▌") + strings.Repeat(" ", len(text)-shown-1))
	}
	return b.String()
}
