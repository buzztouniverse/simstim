package tui

import (
	"image/color"
	"runtime"
	"time"

	"charm.land/lipgloss/v2"
)

// Colour tokens (docs/ux.md 4). They follow what Amp's CLI draws, measured
// cell by cell: nearly every colour is an ANSI palette slot or the terminal's
// own default, so the screens take on the terminal's theme. Secondary text is
// faint (SGR 2), not a grey. Fixed RGB is kept for the brand only.
var (
	colTitle   color.Color = lipgloss.Yellow // titles in borders, and the selection bar
	colKey     color.Color = lipgloss.Blue   // key names and suggestions
	colOnSel   color.Color = lipgloss.Black  // text on the selection bar
	colSuccess color.Color = lipgloss.Green
	colWarning color.Color = lipgloss.Yellow
	colDanger  color.Color = lipgloss.Red
	colBrand   color.Color = lipgloss.Cyan // the brand with 16 colours
)

// brandStops is the one fixed gradient, for the welcome mark and the brand
// name: deep teal to electric cyan. On a light background the teal is
// deeper, so it keeps its contrast. A 256-colour terminal gets the nearest
// indexes; with 16 colours the brand is plain cyan (docs/ux.md 4.2).
func brandStops(dark bool) []color.Color {
	if dark {
		return []color.Color{lipgloss.Color("#0E7C7B"), lipgloss.Color("#00E5FF")}
	}
	return []color.Color{lipgloss.Color("#045A5A"), lipgloss.Color("#0098B8")}
}

// Styles built from the tokens.
var (
	sPlain   = lipgloss.NewStyle()
	sBold    = lipgloss.NewStyle().Bold(true)
	sFaint   = lipgloss.NewStyle().Faint(true)
	sTitle   = lipgloss.NewStyle().Foreground(colTitle).Bold(true)
	sKey     = lipgloss.NewStyle().Foreground(colKey)            // a key in a border hint
	sKeyBold = lipgloss.NewStyle().Foreground(colKey).Bold(true) // a key in a list or a popup
	sLink    = lipgloss.NewStyle().Foreground(colKey)            // what to try next
	sSuccess = lipgloss.NewStyle().Foreground(colSuccess)
	sWarning = lipgloss.NewStyle().Foreground(colWarning)
	sDanger  = lipgloss.NewStyle().Foreground(colDanger)
	sErrHead = lipgloss.NewStyle().Foreground(colDanger).Bold(true) // "Error:"
	sBrand   = lipgloss.NewStyle().Foreground(colBrand)
)

// Glyphs (docs/ux.md 3).
const (
	glyphNeedsInput = "●"
	glyphOnline     = "●"
	glyphIdle       = "○"
	glyphOK         = "✓"
	glyphFail       = "✕"
	glyphOffline    = "◌"
	glyphInfo       = "•"
	glyphSpinStill  = "≈"
	dot             = " · "
)

// The spinner is a small wave, one style everywhere.
var spinFrames = []string{"∼", "≈", "≋", "≈"}

// optKey names the Alt key the way the platform does.
var optKey = func() string {
	if runtime.GOOS == "darwin" {
		return "Opt"
	}
	return "Alt"
}()

// Timings.
const (
	spinEvery      = 200 * time.Millisecond
	heroEvery      = 66 * time.Millisecond // about 15 fps, under the 30 fps cap
	refreshEvery   = 2 * time.Second
	latencyEvery   = 10 * time.Second
	previewDelay   = 120 * time.Millisecond
	toastLife      = 4 * time.Second
	toastErrorLife = 8 * time.Second
	maxBackoff     = 30 * time.Second
)
