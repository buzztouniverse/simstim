// Package termtest makes terminal output for tests and benchmarks. Only
// tests import it.
package termtest

import (
	"bytes"
	"fmt"
)

// AgentOutput is about n bytes of output shaped like a coding agent's
// full-screen redraws: synchronized updates, cursor moves, line erases,
// truecolour and bold text, box drawing, wide characters and OSC 8 links.
// It is the same for the same n.
func AgentOutput(n int) []byte {
	var b bytes.Buffer
	words := []string{"Reading", "internal/host/holder.go", "and", "the", "tests", "✓", "passed", "─────", "日本語", "→"}
	for frame := 0; b.Len() < n; frame++ {
		b.WriteString("\x1b[?2026h\x1b[H")
		for row := 1; row <= 24 && b.Len() < n; row++ {
			fmt.Fprintf(&b, "\x1b[%d;1H\x1b[2K", row)
			r, g, bl := (frame*7+row*13)%256, (row*29)%256, (frame*31)%256
			fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", r, g, bl)
			if row%5 == 0 {
				b.WriteString("\x1b[1m")
			}
			for w := range 9 {
				b.WriteString(words[(frame+row+w)%len(words)])
				b.WriteByte(' ')
			}
			if row%11 == 0 {
				b.WriteString("\x1b]8;;https://example.com/docs\x1b\\link\x1b]8;;\x1b\\")
			}
			b.WriteString("\x1b[0m")
		}
		b.WriteString("\x1b[24;3H\x1b[?2026l")
	}
	return b.Bytes()
}
