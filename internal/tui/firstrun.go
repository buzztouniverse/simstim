package tui

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// The hero (docs/ux.md 2.1): a signal line with a pulse in it, the "flip"
// into a session. With motion on, a ripple runs out through the dots along
// the ramp · • ●, then the screen rests. With motion off it is still.
var heroArt = []string{
	"           ·  ·  ·",
	"     ·  ·  ∿∿∿∿∿∿∿  ·  ·",
	"·  ─────────╯     ╰─────────  ·",
	"     ·  ·  ·  ·  ·  ·  ·",
}

const (
	heroW     = 31
	heroCycle = 90 // frames: about 6 s
	heroPulse = 40 // of which the ripple takes the first 2.6 s
)

// gradient is the brand gradient over n cells, or nil with 16 colours,
// where the mark falls back to the brand slot.
func (m *Model) gradient(n int) []color.Color {
	if !m.richColor || n <= 0 {
		return nil
	}
	return lipgloss.Blend1D(n, brandStops(m.dark)...)
}

func (m *Model) heroLines() []string {
	cols := m.gradient(heroW)
	f := m.hero % heroCycle
	rippling := m.motion && f < heroPulse
	out := make([]string, len(heroArt))
	for y, line := range heroArt {
		var b strings.Builder
		for x, r := range []rune(line) {
			if r == ' ' {
				b.WriteByte(' ')
				continue
			}
			ch := string(r)
			if r == '·' && rippling {
				d := math.Hypot(float64(x-heroW/2)/2.5, float64(y)-1.5)
				switch front := float64(f)*0.35 - d; {
				case front > 0 && front < 0.8, front >= 1.6 && front < 2.4:
					ch = "•"
				case front >= 0.8 && front < 1.6:
					ch = "●"
				}
			}
			st := sBrand
			if cols != nil {
				st = lipgloss.NewStyle().Foreground(cols[min(x, heroW-1)])
			}
			b.WriteString(st.Render(ch))
		}
		// Pad every row to the same width, so centring keeps them aligned.
		out[y] = b.String() + strings.Repeat(" ", max(heroW-len([]rune(line)), 0))
	}
	return out
}

// welcomeTitle is "Welcome to <name>" in the brand gradient, bold.
func (m *Model) welcomeTitle() string {
	text := []rune("Welcome to " + brand.DisplayName)
	cols := m.gradient(len(text))
	if cols == nil {
		return sBold.Render("Welcome to ") + sBrand.Bold(true).Render(brand.DisplayName)
	}
	var b strings.Builder
	for i, r := range text {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(cols[i]).Render(string(r)))
	}
	return b.String()
}

// viewFirstRun is the screen with no hosts (docs/ux.md 2.1), laid out like
// Amp's welcome: the mark on the left and three short lines beside it, or
// the mark above them in a narrow terminal. Errors, notices and the
// shortcuts go below.
func (m *Model) viewFirstRun() string {
	text := []string{
		m.welcomeTitle(),
		welcomeLine([]hint{{"a", "to add a host"}}, 40),
		welcomeLine([]hint{{"ctrl+o", "for commands"}, {"?", "for shortcuts"}}, 40),
	}
	textW := 0
	for _, l := range text {
		textW = max(textW, cells(l))
	}
	mark := m.heroLines()
	var block []string
	left := 0
	if m.width >= heroW+4+textW+4 {
		// Side by side, the text centred on the mark.
		left = max((m.width-(heroW+4+textW))/2, 0)
		off := (len(mark) - len(text)) / 2
		for i := range max(len(mark), off+len(text)) {
			l := strings.Repeat(" ", heroW)
			if i < len(mark) {
				l = mark[i]
			}
			if j := i - off; j >= 0 && j < len(text) {
				l += "    " + text[j]
			}
			block = append(block, strings.Repeat(" ", left)+l)
		}
		left += heroW + 4
	} else {
		for _, l := range mark {
			block = append(block, center(l, m.width))
		}
		block = append(block, "")
		left = max((m.width-textW)/2, 0)
		for _, l := range text {
			block = append(block, strings.Repeat(" ", left)+l)
		}
	}
	var below []string
	room := max(m.width-left-2, 20)
	if m.loadErr != nil {
		msg, _ := errText(m.loadErr)
		below = append(below, errorBlock(msg, "Fix or move the hosts file, then start again.", room)...)
	}
	below = append(below, m.noticeLines(room)...)
	if m.showHelp {
		below = append(below, m.helpLines(min(room, 40))...)
	}
	lines := block
	if len(below) > 0 {
		lines = append(lines, "")
		for _, l := range below {
			lines = append(lines, strings.Repeat(" ", left)+l)
		}
	}
	top := max((m.height-len(block))/2-1, 0)
	if top+len(lines) > m.height {
		top = max(m.height-len(lines), 0)
	}
	out := make([]string, 0, m.height)
	for range top {
		out = append(out, "")
	}
	out = append(out, lines...)
	for len(out) < m.height {
		out = append(out, "")
	}
	return strings.Join(out[:m.height], "\n")
}
