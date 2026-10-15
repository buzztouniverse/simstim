package tui

import (
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/iamlakshya/simstim/internal/brand"
)

// The welcome mark (docs/contributing/ux-spec.md 2.1): three strands of a signal, woven into
// a ribbon that pinches at both ends, drawn in braille so the curves are
// smooth, in the brand gradient. A pulse runs along it, and the strands are
// brighter where they cross. With motion on it flows at 30 fps; with motion
// off it is one still frame.

const (
	markW     = 44
	markH     = 7
	markPulse = 2600 * time.Millisecond // one pulse along the ribbon
	markGrow  = 900 * time.Millisecond  // the ribbon opening out from its middle
)

type strand struct{ amp, wave, speed, phase float64 }

var strands = []strand{
	{amp: 1.0, wave: 1.0, speed: 1.2, phase: 0},
	{amp: 0.8, wave: 1.6, speed: -0.9, phase: 2.2},
	{amp: 0.55, wave: 2.4, speed: 1.7, phase: 4.4},
}

// brailleBit is the dot of a braille cell at column x (0–1) and row y (0–3).
var brailleBit = [4][2]uint{{0, 3}, {1, 4}, {2, 5}, {6, 7}}

// markDots draws the mark's strands at time t on a grid of braille dots,
// opened from the middle to the share grow of its width.
func markDots(t, grow float64) [][]bool {
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
			if math.Abs(u-0.5) > grow/2 {
				prev = -1
				continue // not yet open this far
			}
			env := math.Pow(math.Sin(math.Pi*u), 1.2) * grow
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
	return dots
}

// brailleCell is the braille pattern of the cell at cx, cy, and how many
// of its eight dots are set.
func brailleCell(dots [][]bool, cx, cy int) (pattern, n int) {
	for dy := range 4 {
		for dx := range 2 {
			if dots[cy*4+dy][cx*2+dx] {
				pattern |= 1 << brailleBit[dy][dx]
				n++
			}
		}
	}
	return pattern, n
}

// markLines draws the mark at the current moment.
func (m *Model) markLines() []string {
	t := 0.0
	if m.motion {
		t = m.since(m.epoch).Seconds()
	}
	grow := 1.0
	if m.motion {
		grow = ease(float64(m.since(m.epoch)) / float64(markGrow))
	}
	dots := markDots(t, grow)
	cols := m.gradient(markW)
	smooth := m.smooth()
	back := bg(m)
	pulse := -5 + math.Mod(t/markPulse.Seconds(), 1)*(markW+10)
	out := make([]string, markH)
	for cy := range markH {
		var b strings.Builder
		for cx := range markW {
			r, n := brailleCell(dots, cx, cy)
			if r == 0 {
				b.WriteByte(' ')
				continue
			}
			glyph := string(rune(0x2800 + r)) //nolint:gosec // G115: r is an 8-dot pattern, at most 0xFF
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

// welcomeTitle is the name, letter-spaced, in the brand gradient, bold. With
// motion on it decodes out of braille noise after the ribbon opens.
func (m *Model) welcomeTitle() string {
	spaced := strings.Join(strings.Split(brand.Name, ""), " ")
	text := []rune(m.decode(spaced, m.epoch.Add(markGrow/3)))
	cols := m.gradient(len(text))
	var b strings.Builder
	for i, r := range text {
		st := sBrand.Bold(true)
		if cols != nil {
			st = lipgloss.NewStyle().Foreground(cols[i]).Bold(true)
		}
		b.WriteString(st.Render(string(r)))
	}
	return b.String()
}
