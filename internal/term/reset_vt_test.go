package term

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

// emu wraps an emulator and collects what it writes back (its replies to queries).
type emu struct {
	e   *vt.SafeEmulator
	mu  sync.Mutex
	out bytes.Buffer
}

func newEmu() *emu {
	m := &emu{e: vt.NewSafeEmulator(80, 24)}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := m.e.Read(buf)
			m.mu.Lock()
			m.out.Write(buf[:n])
			m.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	return m
}

func (m *emu) write(b []byte) { _, _ = m.e.Write(b) }

// mode asks the emulator whether a DEC private mode is on (DECRQM). DECRQM
// answers 1 (set) or 3 (permanently set) for on; 0, 2 and 4 all mean off.
// (The emulators are not closed: x/vt v0.1.0 races between Close and Read.)
func (m *emu) mode(n int) string {
	r := m.query(n)
	switch {
	case bytes.Contains([]byte(r), []byte(";1$y")), bytes.Contains([]byte(r), []byte(";3$y")):
		return "on"
	case bytes.Contains([]byte(r), []byte("$y")):
		return "off"
	}
	return r
}

func (m *emu) query(n int) string {
	m.mu.Lock()
	m.out.Reset()
	m.mu.Unlock()
	m.write([]byte(fmt.Sprintf("\x1b[?%d$p", n)))
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		s := m.out.String()
		m.mu.Unlock()
		if bytes.Contains([]byte(s), []byte("$y")) {
			return s
		}
		time.Sleep(time.Millisecond)
	}
	return "no reply"
}

// Sessions that change many modes, the way real agents and editors do.
var sessions = map[string]string{
	"inline agent": "\x1b[>1u\x1b[?2004h\x1b[?1004h\x1b[?25l\x1b[?2026h\x1b[2K\x1b[1Gworking…\x1b[?2026l" +
		"\x1b]11;rgb:1010/1010/1010\x1b\\",
	"full-screen editor": "\x1b[?1049h\x1b[>5u\x1b[?1h\x1b=\x1b[?1000h\x1b[?1002h\x1b[?1006h\x1b[2;20r\x1b[5 q" +
		"\x1b[?7l\x1b[4h\x1b[>4;2m\x1b[H\x1b[2Jsome text",
	"left mid-update": "\x1b[?2026h\x1b[?1003h\x1b[?1049h\x1b[>3u partial",
}

var watchedModes = []int{1, 6, 7, 25, 1000, 1002, 1003, 1004, 1006, 1049, 2004, 2026}

// After a session changes modes, the reset sequence must bring an emulator back
// to the same mode settings as a fresh one, without RIS.
func TestResetRestoresBaseline(t *testing.T) {
	fresh := newEmu()
	base := map[int]string{}
	for _, n := range watchedModes {
		base[n] = fresh.mode(n)
	}
	for name, out := range sessions {
		t.Run(name, func(t *testing.T) {
			e := newEmu()
			tr := NewTracker()
			e.write([]byte(out))
			tr.Feed([]byte(out))
			reset := tr.M.ResetSequence()
			if bytes.Contains(reset, []byte("\x1bc")) {
				t.Fatal("the reset sequence contains RIS")
			}
			e.write(reset)
			for _, n := range watchedModes {
				if got := e.mode(n); got != base[n] {
					t.Errorf("mode %d after reset = %q, want %q", n, got, base[n])
				}
			}
			if e.e.CursorHidden() {
				t.Error("cursor still hidden after reset")
			}
		})
	}
}
