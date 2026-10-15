package tui

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// Motion (docs/ux.md 4.5). Every animation is a function of the clock, so
// any moment can be drawn and tests can stop time. One tick runs while
// something moves, fast or slow as the moving thing needs, and stops when
// nothing does. With motion off, each animation draws one still frame.

const (
	fastFrame = 33 * time.Millisecond  // 30 fps: the welcome mark, glows, fades
	slowFrame = 100 * time.Millisecond // breathing dots and the working glyph

	revealStep  = 22 * time.Millisecond  // between rows as a list appears
	revealFade  = 140 * time.Millisecond // each row's fade in
	noticeIn    = 150 * time.Millisecond
	noticeOut   = 300 * time.Millisecond
	sweepTime   = 520 * time.Millisecond // one lap of the glow when a dialog opens
	glowLap     = 1600 * time.Millisecond
	breathe     = 1600 * time.Millisecond
	shimmerLap  = 1400 * time.Millisecond
	typeEvery   = 38 * time.Millisecond
	lifeEvery   = 140 * time.Millisecond
	glowSpread  = 5.0 // cells either side of the glow's head
	shimmerHalf = 3.0 // cells either side of the shimmer's band
)

type animTickMsg struct{}

// Terminal colours. To blend smoothly the launcher needs the real colours of
// the theme, so it asks the terminal for its default colours (OSC 10 and 11)
// and the 16 palette slots (OSC 4), as Amp does. Until the answers arrive, or
// if they never do, motion uses attributes only: faint, normal and bold.

type termColors struct {
	fg, bg color.Color
	slot   [16]color.Color
}

// queryPalette asks for the 16 palette slots.
func queryPalette() tea.Cmd {
	var b strings.Builder
	for i := range 16 {
		fmt.Fprintf(&b, "\x1b]4;%d;?\x07", i)
	}
	return tea.Raw(b.String())
}

// onOSC reads an OSC 4 reply: "ESC ] 4 ; n ; rgb:rrrr/gggg/bbbb ST".
func (tc *termColors) onOSC(s string) {
	s = strings.TrimPrefix(s, "\x1b]")
	s = strings.TrimSuffix(strings.TrimSuffix(s, "\x07"), "\x1b\\")
	parts := strings.Split(s, ";")
	if len(parts) < 3 || parts[0] != "4" {
		return
	}
	for i := 1; i+1 < len(parts); i += 2 {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 || n > 15 {
			continue
		}
		if c := ansi.XParseColor(parts[i+1]); c != nil {
			tc.slot[n] = c
		}
	}
}

// rgb is a colour with float channels, for blending.
type rgb struct{ r, g, b float64 }

func toRGB(c color.Color) rgb {
	r, g, b, _ := c.RGBA()
	return rgb{float64(r >> 8), float64(g >> 8), float64(b >> 8)}
}

func (c rgb) color() color.Color {
	cl := func(v float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(255, v)))) }
	return color.RGBA{cl(c.r), cl(c.g), cl(c.b), 255}
}

// mix is a towards b by t (0 gives a, 1 gives b).
func mix(a, b rgb, t float64) rgb {
	t = math.Max(0, math.Min(1, t))
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

// resolve turns a cell colour into RGB: nil is the default foreground or
// background, slots come from the terminal's answers.
func (tc *termColors) resolve(c color.Color, fg bool) (rgb, bool) {
	switch v := c.(type) {
	case nil:
		def := tc.bg
		if fg {
			def = tc.fg
		}
		if def == nil {
			return rgb{}, false
		}
		return toRGB(def), true
	case ansi.BasicColor:
		if s := tc.slot[int(v)&15]; s != nil {
			return toRGB(s), true
		}
		return rgb{}, false
	case ansi.IndexedColor:
		if v < 16 {
			if s := tc.slot[v]; s != nil {
				return toRGB(s), true
			}
			return rgb{}, false
		}
	}
	return toRGB(c), true
}

// smooth reports whether frames can blend colours: motion on, truecolour,
// and the terminal told us its default colours.
func (m *Model) smooth() bool {
	return m.motion && m.truecolor && m.colors.fg != nil && m.colors.bg != nil
}

// since is how long ago t was, never negative.
func (m *Model) since(t time.Time) time.Duration {
	if t.IsZero() {
		return time.Hour
	}
	return max(m.now().Sub(t), 0)
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// ease is a smooth start and stop.
func ease(t float64) float64 { t = clamp01(t); return t * t * (3 - 2*t) }

// Frame scheduling.

// frameNeed is how soon the next frame is needed, or 0 when nothing moves.
func (m *Model) frameNeed() time.Duration {
	if !m.motion {
		return 0
	}
	switch {
	case len(m.hosts) == 0 && m.modal == nil:
		return fastFrame // the welcome mark
	case m.since(m.revealAt) < revealFade+revealStep*20,
		m.since(m.modalAt) < sweepTime,
		m.glowing(),
		m.noticeMoving():
		return fastFrame
	case m.needSpin():
		return slowFrame
	}
	return 0
}

// startTimers keeps one frame tick running while anything moves. Idle screens
// never repaint.
func (m *Model) startTimers() tea.Cmd {
	d := m.frameNeed()
	if d == 0 || m.ticking {
		return nil
	}
	m.ticking = true
	return m.after(d, animTickMsg{})
}

// Canvas effects. They run on the drawn canvas, cell by cell, so they work on
// any part of the screen whatever drew it.

// isLine reports whether a cell is part of a frame's line.
func isLine(s string) bool {
	return strings.ContainsAny(s, "╭╮╰╯─│") && len([]rune(s)) == 1
}

// glow lights a frame's border with a soft head that travels round it. pos
// is where the head is, 0 to 1 round the frame from the top left; strength
// fades it out. The rest of the border dims while it glows, as a busy frame.
type glow struct {
	x, y, w, h int
	pos        float64
	strength   float64
}

func (m *Model) applyGlow(c *lipgloss.Canvas, g glow) {
	if g.w < 2 || g.h < 2 || g.strength <= 0 {
		return
	}
	per := 2*(g.w-1) + 2*(g.h-1)
	head := g.pos * float64(per)
	at := func(i int) (int, int) {
		switch {
		case i < g.w-1:
			return g.x + i, g.y
		case i < g.w-1+g.h-1:
			return g.x + g.w - 1, g.y + i - (g.w - 1)
		case i < 2*(g.w-1)+g.h-1:
			return g.x + g.w - 1 - (i - (g.w - 1 + g.h - 1)), g.y + g.h - 1
		default:
			return g.x, g.y + g.h - 1 - (i - (2*(g.w-1) + g.h - 1))
		}
	}
	smooth := m.smooth()
	var base, hi rgb
	if smooth {
		fg, _ := m.colors.resolve(nil, true)
		bg, _ := m.colors.resolve(nil, false)
		base = mix(fg, bg, 0.55)
		hi = rgb{255, 255, 255}
		if !m.dark {
			hi = mix(fg, rgb{}, 0.5)
		}
	}
	for i := range per {
		x, y := at(i)
		cell := c.CellAt(x, y)
		if cell == nil || !isLine(cell.Content) {
			continue
		}
		d := math.Abs(float64(i) - head)
		d = math.Min(d, float64(per)-d)
		k := g.strength * math.Exp(-(d/glowSpread)*(d/glowSpread))
		if smooth {
			col := base
			if cell.Style.Fg != nil { // a tone keeps its colour under the glow
				if t, ok := m.colors.resolve(cell.Style.Fg, true); ok {
					col = mix(t, bg(m), 0.35)
				}
			}
			cell.Style.Fg = mix(col, hi, k).color()
			cell.Style.Attrs &^= uv.AttrFaint
			continue
		}
		switch {
		case k > 0.55:
			cell.Style.Fg = lipgloss.BrightWhite
			cell.Style.Attrs = cell.Style.Attrs&^uv.AttrFaint | uv.AttrBold
		case k > 0.2:
			cell.Style.Attrs &^= uv.AttrFaint
		default:
			cell.Style.Attrs |= uv.AttrFaint
		}
	}
}

func bg(m *Model) rgb { c, _ := m.colors.resolve(nil, false); return c }

// fade blends a rectangle towards the background: alpha 1 leaves it as it
// is, 0 hides it. Without blends, a cell shows once alpha passes one half.
func (m *Model) fade(c *lipgloss.Canvas, x, y, w, h int, alpha float64) {
	if alpha >= 1 {
		return
	}
	smooth := m.smooth()
	back := bg(m)
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			cell := c.CellAt(xx, yy)
			if cell == nil {
				continue
			}
			if !smooth {
				if alpha < 0.5 {
					*cell = uv.EmptyCell
				}
				continue
			}
			fg, ok := m.colors.resolve(cell.Style.Fg, true)
			if !ok {
				if alpha < 0.5 {
					*cell = uv.EmptyCell
				}
				continue
			}
			if cell.Style.Attrs&uv.AttrFaint != 0 {
				fg = mix(fg, back, 0.5)
				cell.Style.Attrs &^= uv.AttrFaint
			}
			cell.Style.Fg = mix(back, fg, alpha).color()
			if cell.Style.Bg != nil {
				if b, ok := m.colors.resolve(cell.Style.Bg, false); ok {
					cell.Style.Bg = mix(back, b, alpha).color()
				}
			}
		}
	}
}

// Text effects, drawn into strings.

// shimmer draws faint text with a brighter band that sweeps across it, for
// things in progress: "Connecting…", "Starting claude on homelab…".
func (m *Model) shimmer(text string) string {
	if !m.motion {
		return sFaint.Render(text)
	}
	runes := []rune(text)
	n := float64(len(runes))
	lap := float64(m.since(m.epoch)%shimmerLap) / float64(shimmerLap)
	center := -shimmerHalf + lap*(n+2*shimmerHalf)
	smooth := m.smooth()
	var fg, back rgb
	if smooth {
		fg, _ = m.colors.resolve(nil, true)
		back = bg(m)
	}
	var b strings.Builder
	for i, r := range runes {
		d := math.Abs(float64(i) - center)
		k := math.Max(0, 1-d/shimmerHalf)
		switch {
		case smooth:
			b.WriteString(lipgloss.NewStyle().Foreground(mix(mix(fg, back, 0.5), fg, k).color()).Render(string(r)))
		case k > 0.3:
			b.WriteString(string(r))
		default:
			b.WriteString(sFaint.Render(string(r)))
		}
	}
	return b.String()
}

// breath is the colour of a dot that needs the user: it breathes between
// full and dim. Without blends it dims on the out-breath.
func (m *Model) breath(st lipgloss.Style) lipgloss.Style {
	if !m.motion {
		return st
	}
	phase := float64(m.since(m.epoch)%breathe) / float64(breathe)
	k := 0.5 + 0.5*math.Cos(2*math.Pi*phase) // 1 at the start of each breath
	if m.smooth() {
		if c, ok := m.colors.resolve(st.GetForeground(), true); ok {
			return st.Foreground(mix(bg(m), c, 0.35+0.65*k).color())
		}
	}
	if k < 0.35 {
		return st.Faint(true)
	}
	return st
}

// Spinners.

// spinner is the wave for things in progress, or a still one with motion off.
func (m *Model) spinner() string {
	if !m.motion {
		return glyphSpinStill
	}
	return spinFrames[int(m.since(m.epoch)/spinEvery)%len(spinFrames)]
}

// lifeFrames is a loop of braille cells running a tiny Game of Life on the
// cell's eight dots: the glyph of a working session. When the dots die out or
// freeze, a new random pattern starts.
var lifeFrames = func() []string {
	// The eight dots as a 2×4 grid, wrapping at the edges.
	bit := [4][2]uint{{0, 3}, {1, 4}, {2, 5}, {6, 7}}
	alive := func(s uint8, x, y int) bool {
		x, y = (x+2)%2, (y+4)%4
		return s&(1<<bit[y][x]) != 0
	}
	step := func(s uint8) uint8 {
		var n uint8
		for y := range 4 {
			for x := range 2 {
				c := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if (dx != 0 || dy != 0) && alive(s, x+dx, y+dy) {
							c++
						}
					}
				}
				if c == 3 || c == 2 && alive(s, x, y) {
					n |= 1 << bit[y][x]
				}
			}
		}
		return n
	}
	seed := uint32(0x9E3779B9)
	rnd := func() uint8 { seed ^= seed << 13; seed ^= seed >> 17; seed ^= seed << 5; return uint8(seed) }
	var out []string
	s := rnd() | 0x11
	for len(out) < 48 {
		out = append(out, string(rune(0x2800+int(s))))
		n := step(s)
		if n == 0 || n == s {
			n = rnd() | 0x24
		}
		s = n
	}
	return out
}()

// working is the glyph of a working session: its own Game of Life, offset by
// its ID so two sessions never move in step.
func (m *Model) working(id string) string {
	if !m.motion {
		return glyphSpinStill
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	i := int(m.since(m.epoch)/lifeEvery) + int(h.Sum32()%uint32(len(lifeFrames)))
	return lifeFrames[i%len(lifeFrames)]
}

// Moving parts.

// glowing reports whether a frame on screen is busy: a host still
// connecting, a session starting, the wizard at work.
func (m *Model) glowing() bool {
	if b, ok := m.modal.(interface{ busy() bool }); ok && b.busy() {
		return true
	}
	if m.modal != nil || len(m.hosts) == 0 {
		return false
	}
	if m.screen == scrSessions {
		if h := m.selectedHost(); h != nil {
			return h.status == stConnecting
		}
	}
	for _, h := range m.hosts {
		if h.loaded {
			return false
		}
	}
	return true // nothing has answered yet
}

// glowAt is the glow for a frame drawn at x, y, w×h: a lap when a dialog
// opens, then a steady travel while busy.
func (m *Model) glowAt(x, y, w, h int, busy bool, opened time.Time) (glow, bool) {
	if !m.motion {
		return glow{}, false
	}
	g := glow{x: x, y: y, w: w, h: h}
	if s := m.since(opened); s < sweepTime {
		p := float64(s) / float64(sweepTime)
		g.pos, g.strength = ease(p), 1-p*p
		return g, true
	}
	if busy {
		g.pos = float64(m.since(m.epoch)%glowLap) / float64(glowLap)
		g.strength = 1
		return g, true
	}
	return glow{}, false
}

func (m *Model) noticeMoving() bool {
	now := m.now()
	for _, t := range m.toasts.shown {
		if now.Sub(t.at) < noticeIn || t.until.Sub(now) < noticeOut {
			return true
		}
	}
	return false
}

// noticeAlpha is how visible a notice is: it fades in, and out at the end.
func (m *Model) noticeAlpha(t toast) float64 {
	if !m.motion {
		return 1
	}
	now := m.now()
	in := clamp01(float64(now.Sub(t.at)) / float64(noticeIn))
	out := clamp01(float64(t.until.Sub(now)) / float64(noticeOut))
	return math.Min(ease(in), ease(out))
}

// rowAlpha is how far row i of a list has faded in after the list appeared.
func (m *Model) rowAlpha(i int) float64 {
	if !m.motion {
		return 1
	}
	s := m.since(m.revealAt) - time.Duration(i)*revealStep
	return ease(float64(s) / float64(revealFade))
}
