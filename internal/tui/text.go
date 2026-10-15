package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// cells is the width of s in terminal cells, ignoring escape sequences.
func cells(s string) int { return ansi.StringWidth(s) }

// trunc shortens s to at most w cells, ending with "…" when it cuts.
func trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if cells(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "…")
}

// fit makes s exactly w cells wide: cut with "…" or padded with spaces.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = trunc(s, w)
	if n := cells(s); n < w {
		s += strings.Repeat(" ", w-n)
	}
	return s
}

// fitRight is fit, aligned to the right.
func fitRight(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = trunc(s, w)
	if n := cells(s); n < w {
		s = strings.Repeat(" ", w-n) + s
	}
	return s
}

// center places s in the middle of w cells.
func center(s string, w int) string {
	s = trunc(s, w)
	n := cells(s)
	if n >= w {
		return s
	}
	left := (w - n) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", w-n-left)
}

// truncPath shortens a path in the middle, keeping its start and its last
// part: "~/src/…/auth".
func truncPath(p string, w int) string {
	if w <= 0 {
		return ""
	}
	if cells(p) <= w {
		return p
	}
	parts := strings.Split(p, "/")
	if len(parts) > 2 {
		for keep := len(parts) - 2; keep >= 1; keep-- {
			s := strings.Join(parts[:keep], "/") + "/…/" + parts[len(parts)-1]
			if cells(s) <= w {
				return s
			}
		}
	}
	r := []rune(p)
	if w < 3 {
		return string(r[len(r)-w:])
	}
	head := (w - 1) / 2
	tail := w - 1 - head
	return string(r[:head]) + "…" + string(r[len(r)-tail:])
}

// seg is a piece of a row with its style.
type seg struct {
	s  string
	st lipgloss.Style
}

// row renders segments, each with its own style. With reverse set, every
// segment is drawn in reverse video, so the whole row reads as selected.
func row(reverse bool, segs ...seg) string {
	var b strings.Builder
	for _, sg := range segs {
		st := sg.st
		if reverse {
			st = st.Reverse(true)
		}
		b.WriteString(st.Render(sg.s))
	}
	return b.String()
}

// plural picks the word for n.
func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// fmtAge is a short age: 14s, 2m, 1h, 3d.
func fmtAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// fmtLatency is a short round-trip time: 9ms, 1.2s.
func fmtLatency(d time.Duration) string {
	switch {
	case d <= 0:
		return ""
	case d < time.Second:
		ms := d.Milliseconds()
		if ms < 1 {
			ms = 1
		}
		return fmt.Sprintf("%dms", ms)
	default:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
}

// fmtSize is a size in binary units: 310 MiB, 6.1 GiB.
func fmtSize(b int64) string {
	const mib, gib = 1 << 20, 1 << 30
	switch {
	case b >= gib:
		return trimZero(fmt.Sprintf("%.1f", float64(b)/gib)) + " GiB"
	case b >= mib:
		return fmt.Sprintf("%d MiB", b/mib)
	default:
		return fmt.Sprintf("%d KiB", b>>10)
	}
}

// fmtMemory is the memory line of a session: "6.1 of 8 GiB".
func fmtMemory(cur, limit int64) string {
	switch {
	case cur <= 0 && limit <= 0:
		return ""
	case limit <= 0:
		return fmtSize(cur)
	case cur <= 0:
		return "limit " + fmtSize(limit)
	}
	c, l := fmtSize(cur), fmtSize(limit)
	if strings.HasSuffix(c, " GiB") && strings.HasSuffix(l, " GiB") {
		return strings.TrimSuffix(c, " GiB") + " of " + l
	}
	return c + " of " + l
}

func trimZero(s string) string { return strings.TrimSuffix(s, ".0") }

// fuzzy reports whether every letter of q appears in s in order, and scores
// the match: runs of letters and word starts count more, as do short strings.
func fuzzy(q, s string) (int, bool) {
	q = strings.ToLower(strings.Join(strings.Fields(q), ""))
	if q == "" {
		return 0, true
	}
	qr, sr := []rune(q), []rune(strings.ToLower(s))
	score, qi, last := 0, 0, -2
	for i, r := range sr {
		if qi == len(qr) {
			break
		}
		if r != qr[qi] {
			continue
		}
		score++
		if last == i-1 {
			score += 3
		}
		if i == 0 || !unicode.IsLetter(sr[i-1]) && !unicode.IsDigit(sr[i-1]) {
			score += 2
		}
		last = i
		qi++
	}
	if qi < len(qr) {
		return 0, false
	}
	return score*8 - len(sr), true
}

// errText splits an error into its message and its "Try:" hint.
func errText(err error) (msg, hint string) {
	if err == nil {
		return "", ""
	}
	if ue, ok := asUserError(err); ok {
		return ue.Msg, ue.Hint
	}
	return err.Error(), ""
}

// sentence makes a message start with a capital letter and end with a stop.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	s = string(r)
	if !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, "?") && !strings.HasSuffix(s, ")") {
		s += "."
	}
	return s
}

// wrap breaks s into lines of at most w cells, at spaces where it can.
func wrap(s string, w int) []string {
	if w <= 0 {
		return nil
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		out = append(out, strings.Split(ansi.Wrap(para, w, ""), "\n")...)
	}
	return out
}
