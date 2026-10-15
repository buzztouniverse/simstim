package host

import (
	"errors"
	"testing"
)

// When a holder stops before its socket answers, the user reads the last
// line of its log as text: the message and the error, without the time and
// the level.
func TestLogText(t *testing.T) {
	var logs lineLog
	log := logs.logger()
	log.Error("listen failed", "err", errors.New(`listen unix /tmp/s.sock: bind: "address in use"`))
	log.Info("holder starting", "session", "abc", "version", "1.0")
	lines := logs.get()
	for i, want := range []string{`listen failed: listen unix /tmp/s.sock: bind: "address in use"`, "holder starting"} {
		if got := logText(lines[i]); got != want {
			t.Errorf("logText(%q) = %q, want %q", lines[i], got, want)
		}
	}
	// A line an older helper wrote, or one cut short, comes back as it is.
	for _, line := range []string{"2026/10/15 12:00:00.123456 listen: address in use", `time=x msg="cut sho`} {
		if got := logText(line); got != line {
			t.Errorf("logText(%q) = %q", line, got)
		}
	}
}
