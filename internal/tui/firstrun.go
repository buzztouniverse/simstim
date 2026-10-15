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

// gradient is the brand gradient over n cells, or nil without truecolour,
// where the hero falls back to the accent token.
func (m *Model) gradient(n int) []color.Color {
	if !m.truecolor || n <= 0 {
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
			st := sAccent
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

// brandTitle draws the product name in the brand gradient.
func (m *Model) brandTitle() string {
	name := []rune(brand.DisplayName)
	cols := m.gradient(len(name))
	if cols == nil {
		return sAccent.Bold(true).Render(brand.DisplayName)
	}
	var b strings.Builder
	for i, r := range name {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(cols[i]).Render(string(r)))
	}
	return b.String()
}

// viewFirstRun is the screen with no hosts: the hero, one plain line, and one
// line of at most three hints.
func (m *Model) viewFirstRun() string {
	w := m.width
	var lines []string
	for _, l := range m.heroLines() {
		lines = append(lines, center(l, w))
	}
	lines = append(lines, "", center(m.brandTitle(), w), center(sMuted.Render(brand.Tagline), w), "")
	if m.loadErr != nil {
		msg, _ := errText(m.loadErr)
		for _, l := range wrap(sentence(msg), min(w-4, 72)) {
			lines = append(lines, center(sDanger.Render(l), w))
		}
		lines = append(lines, center(sPrimary.Render("Try:")+" fix or move the hosts file, then start again.", w))
	} else {
		lines = append(lines, center("No hosts yet. Press a to add one.", w))
	}
	lines = append(lines, "", center(hintLine([]hint{{"a", "add host"}, {"ctrl+o", "commands"}, {"?", "shortcuts"}}, w-2), w))
	top := max((m.height-len(lines))/2, 0)
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
