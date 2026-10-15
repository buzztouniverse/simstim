package tui

import (
	"image/color"
	"time"

	"charm.land/lipgloss/v2"
)

// Colour tokens (docs/ux.md 4.1). Each maps to an ANSI palette slot, so the
// screens follow the terminal's own theme with no config.
var (
	colMuted       color.Color = lipgloss.BrightBlack
	colPrimary     color.Color = lipgloss.Blue
	colSecondary   color.Color = lipgloss.Cyan
	colAccent      color.Color = lipgloss.Magenta
	colBorder      color.Color = lipgloss.BrightBlack
	colSuccess     color.Color = lipgloss.Green
	colWarning     color.Color = lipgloss.Yellow
	colDestructive color.Color = lipgloss.Red
	colInfo        color.Color = lipgloss.BrightBlue
)

// brandStops is the one fixed gradient, kept for the hero and the first-run
// title only: deep teal to electric cyan. On a light background the teal is
// deeper, so the hero keeps its contrast.
func brandStops(dark bool) []color.Color {
	if dark {
		return []color.Color{lipgloss.Color("#0E7C7B"), lipgloss.Color("#00E5FF")}
	}
	return []color.Color{lipgloss.Color("#045A5A"), lipgloss.Color("#0098B8")}
}

// Styles built from the tokens.
var (
	sBold      = lipgloss.NewStyle().Bold(true)
	sMuted     = lipgloss.NewStyle().Foreground(colMuted)
	sPrimary   = lipgloss.NewStyle().Foreground(colPrimary)
	sKey       = lipgloss.NewStyle().Foreground(colPrimary).Bold(true)
	sSecondary = lipgloss.NewStyle().Foreground(colSecondary)
	sAccent    = lipgloss.NewStyle().Foreground(colAccent)
	sBorder    = lipgloss.NewStyle().Foreground(colBorder)
	sSuccess   = lipgloss.NewStyle().Foreground(colSuccess)
	sWarning   = lipgloss.NewStyle().Foreground(colWarning)
	sDanger    = lipgloss.NewStyle().Foreground(colDestructive)
	sInfo      = lipgloss.NewStyle().Foreground(colInfo)
	sLabel     = lipgloss.NewStyle().Foreground(colMuted).Bold(true)
	sPlain     = lipgloss.NewStyle()
)

// Glyphs (docs/ux.md 2.3).
const (
	glyphNeedsInput = "●"
	glyphIdle       = "○"
	glyphOK         = "✓"
	glyphFail       = "✕"
	glyphOffline    = "◌"
	glyphPointer    = "▸"
	glyphInfo       = "•"
	glyphSpinStill  = "⠶"
	dot             = " · "
)

// The spinner is our own braille "scan", one style everywhere.
var spinFrames = []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}

// Timings.
const (
	spinEvery      = 120 * time.Millisecond
	heroEvery      = 66 * time.Millisecond // about 15 fps, under the 30 fps cap
	refreshEvery   = 2 * time.Second
	latencyEvery   = 10 * time.Second
	previewDelay   = 120 * time.Millisecond
	toastLife      = 3 * time.Second
	toastErrorLife = 6 * time.Second
	quitWindow     = time.Second
	maxBackoff     = 30 * time.Second
)
