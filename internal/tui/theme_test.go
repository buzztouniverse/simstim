package tui

import (
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

// The launcher in every colour profile a terminal can have (docs/ux.md 4.1):
// truecolour, 256 colours, 16 colours, NO_COLOR (attributes only) and no
// terminal at all. Bubble Tea turns the view into the terminal's profile the
// same way, through colorprofile.

// themeScreens are the screens checked in each profile and theme.
var themeScreens = []struct {
	name  string
	setup func(t *testing.T) *harness
}{
	{"first run", func(t *testing.T) *harness { return newHarness(t, newFakeBackend(), 80, 24) }},
	{"hosts", func(t *testing.T) *harness { return home(t, 80, 24) }},
	{"sessions", func(t *testing.T) *harness { return sessions(t, 80, 24) }},
	{"delete sheet", func(t *testing.T) *harness {
		hs := sessions(t, 80, 24)
		hs.key("d")
		return hs
	}},
	{"commands", func(t *testing.T) *harness {
		hs := sessions(t, 80, 24)
		hs.key("ctrl+o")
		return hs
	}},
}

// inProfile renders the view as a terminal with profile p receives it.
func inProfile(t *testing.T, s string, p colorprofile.Profile) string {
	t.Helper()
	var b strings.Builder
	w := &colorprofile.Writer{Forward: &b, Profile: p}
	if _, err := w.WriteString(s); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

var sgrRe = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)

// sgrUse sorts the SGR parameters in s: plain attributes, 16-colour,
// 256-colour and RGB colours.
type sgrUse struct {
	attrs           map[int]bool
	basic, idx, rgb int
	rgbs            []color.RGBA
}

func sgrs(s string) sgrUse {
	u := sgrUse{attrs: map[int]bool{}}
	for _, m := range sgrRe.FindAllStringSubmatch(s, -1) {
		ps := strings.FieldsFunc(m[1], func(r rune) bool { return r == ';' || r == ':' })
		for i := 0; i < len(ps); i++ {
			n, _ := strconv.Atoi(ps[i])
			switch {
			case (n == 38 || n == 48 || n == 58) && i+1 < len(ps) && ps[i+1] == "5":
				u.idx++
				i += 2
			case (n == 38 || n == 48 || n == 58) && i+1 < len(ps) && ps[i+1] == "2":
				if i+4 < len(ps) {
					r, _ := strconv.Atoi(ps[i+2])
					g, _ := strconv.Atoi(ps[i+3])
					b, _ := strconv.Atoi(ps[i+4])
					u.rgbs = append(u.rgbs, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
				}
				u.rgb++
				i += 4
			case n >= 30 && n <= 37, n >= 40 && n <= 47, n >= 90 && n <= 97, n >= 100 && n <= 107, n == 39, n == 49:
				u.basic++
			default:
				u.attrs[n] = true
			}
		}
	}
	return u
}

func TestColourProfiles(t *testing.T) {
	for _, sc := range themeScreens {
		for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI256, colorprofile.ANSI, colorprofile.Ascii, colorprofile.NoTTY} {
			t.Run(sc.name+"/"+p.String(), func(t *testing.T) {
				hs := sc.setup(t)
				hs.m.motion = false
				hs.send(tea.ColorProfileMsg{Profile: p})
				hs.send(tea.BackgroundColorMsg{Color: color.RGBA{0x0B, 0x0E, 0x14, 0xff}})
				view := hs.m.render()
				out := inProfile(t, view, p)
				u := sgrs(out)
				// The words are the same in every profile.
				if ansi.Strip(out) != ansi.Strip(view) {
					t.Fatalf("the text changed in %v", p)
				}
				switch p {
				case colorprofile.TrueColor:
					if u.rgb == 0 && sc.name != "delete sheet" && sc.name != "commands" {
						t.Errorf("no RGB colour: the brand gradient is missing")
					}
				case colorprofile.ANSI256:
					if u.rgb > 0 {
						t.Errorf("%d RGB colours sent to a 256-colour terminal", u.rgb)
					}
				case colorprofile.ANSI:
					if u.rgb > 0 || u.idx > 0 {
						t.Errorf("%d RGB and %d indexed colours sent to a 16-colour terminal", u.rgb, u.idx)
					}
					if u.basic == 0 {
						t.Errorf("no colour at all with 16 colours")
					}
				case colorprofile.Ascii:
					if u.rgb+u.idx+u.basic > 0 {
						t.Errorf("colours sent with NO_COLOR: %d RGB, %d indexed, %d basic", u.rgb, u.idx, u.basic)
					}
					if !u.attrs[1] && !u.attrs[2] {
						t.Errorf("neither bold nor faint with NO_COLOR, so nothing stands out")
					}
				case colorprofile.NoTTY:
					if strings.Contains(out, "\x1b[") {
						t.Errorf("escape sequences sent with no terminal")
					}
				}
			})
		}
	}
}

// Without colour, the selected row and each state still differ by glyph.
func TestNoColourKeepsMeaning(t *testing.T) {
	hs := sessions(t, 80, 24)
	hs.m.motion = false
	hs.send(tea.ColorProfileMsg{Profile: colorprofile.Ascii})
	out := ansi.Strip(inProfile(t, hs.m.render(), colorprofile.Ascii))
	for _, g := range []string{glyphMarker, glyphNeedsInput, glyphIdle, glyphOK} {
		if !strings.Contains(out, g) {
			t.Errorf("glyph %q is missing without colour", g)
		}
	}
}

// NO_COLOR picks the attributes-only profile, and stills the motion.
func TestNoColourEnv(t *testing.T) {
	if p := colorprofile.Env([]string{"TERM=xterm-256color", "NO_COLOR=1"}); p != colorprofile.Ascii {
		t.Errorf("NO_COLOR gives profile %v, want Ascii", p)
	}
	env := map[string]string{"NO_COLOR": "1"}
	if motionOn("auto", func(k string) string { return env[k] }) {
		t.Error("motion is on with NO_COLOR")
	}
}

// The fixed colours, the brand gradient, keep a contrast of at least 3:1 on
// the background of a dark and of a light theme.
func TestThemeContrast(t *testing.T) {
	for _, bg := range []color.RGBA{
		{0x0B, 0x0E, 0x14, 0xff}, // the reference dark theme
		{0x00, 0x00, 0x00, 0xff},
		{0xFF, 0xFF, 0xFF, 0xff},
		{0xFA, 0xFA, 0xFA, 0xff}, // a light theme's off-white
	} {
		for _, sc := range themeScreens[:3] {
			hs := sc.setup(t)
			hs.m.motion = false
			hs.send(tea.ColorProfileMsg{Profile: colorprofile.TrueColor})
			hs.send(tea.BackgroundColorMsg{Color: bg})
			u := sgrs(hs.m.render())
			if len(u.rgbs) == 0 {
				t.Fatalf("%s on %v: no RGB colours to check", sc.name, bg)
			}
			for _, c := range u.rgbs {
				if r := contrast(c, bg); r < 3 {
					t.Errorf("%s on %v: %v has contrast %.2f, want 3 or more", sc.name, bg, c, r)
				}
			}
		}
	}
}

// contrast is the WCAG contrast ratio of two colours.
func contrast(a, b color.RGBA) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func luminance(c color.RGBA) float64 {
	ch := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}
