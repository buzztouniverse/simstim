package host

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// These tests run the Go side of the shadow terminal against a fake child,
// a shell script, so they need no Zig. The fake writes the frames it is
// given first, then echoes every frame it reads, so each request is answered
// with its own payload, and keeps a copy of its input in a log.

// fakeVT writes the fake child and returns its path and its input log.
func fakeVT(t *testing.T, first ...[]byte) (path, log string) {
	t.Helper()
	dir := t.TempDir()
	path, log = filepath.Join(dir, "fake-vt"), filepath.Join(dir, "input")
	var pre strings.Builder
	for _, f := range first {
		for _, b := range f {
			fmt.Fprintf(&pre, "\\%03o", b)
		}
	}
	script := "#!/bin/sh\necho \"fake shadow terminal $*\" >&2\nprintf '" + pre.String() + "'\nexec tee -a " + log + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path, log
}

// frames decodes a log of frames.
func frames(t *testing.T, b []byte) []string {
	t.Helper()
	var out []string
	for len(b) > 0 {
		if len(b) < 5 {
			t.Fatalf("a torn frame header: %q", b)
		}
		n := int(binary.LittleEndian.Uint32(b[1:5]))
		if len(b) < 5+n {
			t.Fatalf("a torn frame: %q", b)
		}
		out = append(out, fmt.Sprintf("%c%q", b[0], b[5:5+n]))
		b = b[5+n:]
	}
	return out
}

// childFrames waits until the child's input log holds n frames.
func childFrames(t *testing.T, log string, n int) []string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		b, _ := os.ReadFile(log)
		if f := frames(t, b); len(f) >= n || time.Now().After(deadline) {
			return f
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestVTFramesWithAFakeChild(t *testing.T) {
	bell := vtFrame(vtFrameEvent, []byte(`{"type":"bell","value":-1}`))
	reply := vtFrame(vtFrameWrite, []byte("\x1b[?62;22c"))
	path, log := fakeVT(t, bell, reply)
	var mu sync.Mutex
	var events []VTEvent
	var writes, logs []string
	v, err := StartVT(VTOptions{
		Path: path, Cols: 100, Rows: 30, Scrollback: 500,
		OnEvent: func(e VTEvent) { mu.Lock(); events = append(events, e); mu.Unlock() },
		OnWrite: func(p []byte) { mu.Lock(); writes = append(writes, string(p)); mu.Unlock() },
		Logf:    func(f string, a ...any) { mu.Lock(); logs = append(logs, fmt.Sprintf(f, a...)); mu.Unlock() },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	v.Feed(0, []byte("hello"))
	v.Feed(5, nil)
	v.Resize(120, 40, 960, 640)
	v.Resize(0, 40, 0, 0) // ignored
	v.AnswerQueries(true)
	v.SetProfile(VTProfile{XTVersion: "test 1.0"})
	if err := v.sync(); err != nil {
		t.Fatal(err)
	}
	snap, upTo, err := v.Snapshot(90, 20, 50, 5*time.Second)
	if err != nil || upTo != 5 {
		t.Fatalf("snapshot = %q, %d, %v", snap, upTo, err)
	}
	// The fake echoes the request: cols, rows and the scrollback lines.
	if want := []byte{90, 0, 20, 0, 50, 0, 0, 0}; !bytes.Equal(snap, want) {
		t.Errorf("snapshot request = %v, want %v", snap, want)
	}
	if _, _, err := v.Snapshot(0, 20, 50, time.Second); err == nil {
		t.Error("a snapshot of no size worked")
	}
	if d, err := v.dump(7, 5*time.Second); err != nil || !bytes.Equal(d, []byte{7, 0, 0, 0}) {
		t.Errorf("dump = %v, %v", d, err)
	}

	want := []string{
		`R"d\x00\x1e\x00\x00\x00\x00\x00"`, // 100 by 30, first
		`Q"\x00"`,                          // no answers yet
		`D"hello"`,
		`R"x\x00(\x00\xc0\x03\x80\x02"`, // 120 by 40, 960 by 640 pixels
		`Q"\x01"`,
		`P"{\"xtversion\":\"test 1.0\"}"`,
		`Y""`,
		`S"Z\x00\x14\x002\x00\x00\x00"`,
		`T"\a\x00\x00\x00"`,
	}
	if got := childFrames(t, log, len(want)); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("the child got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 1 || events[0].Type != "bell" || len(writes) != 1 || writes[0] != "\x1b[?62;22c" {
		t.Errorf("events %+v, writes %q", events, writes)
	}
	if len(logs) == 0 || logs[0] != "shadow terminal: fake shadow terminal --cols 100 --rows 30 --scrollback 500" {
		t.Errorf("logs = %q", logs)
	}
}

func TestVTRestartsWithBackoff(t *testing.T) {
	old := vtBackoff
	vtBackoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond}
	t.Cleanup(func() { vtBackoff = old })
	path, log := fakeVT(t)
	var mu sync.Mutex
	hist := []byte("history")
	var logs []string
	logged := func(s string) bool {
		mu.Lock()
		defer mu.Unlock()
		for _, l := range logs {
			if strings.Contains(l, s) {
				return true
			}
		}
		return false
	}
	v, err := StartVT(VTOptions{
		Path:    path,
		History: func() ([]byte, int64) { mu.Lock(); defer mu.Unlock(); return bytes.Clone(hist), 10 },
		Logf:    func(f string, a ...any) { mu.Lock(); logs = append(logs, fmt.Sprintf(f, a...)); mu.Unlock() },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	if err := v.sync(); err != nil {
		t.Fatal(err)
	}
	pid := func() int {
		v.mu.Lock()
		defer v.mu.Unlock()
		if v.child == nil {
			return 0
		}
		return v.child.cmd.Process.Pid
	}
	first := pid()
	// The child crashes; output that arrives meanwhile is not lost.
	if err := syscall.Kill(first, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	hist = append(hist, " more"...)
	mu.Unlock()
	v.Feed(17, []byte(" more"))
	deadline := time.Now().Add(5 * time.Second)
	for p := pid(); p == 0 || p == first; p = pid() {
		if time.Now().After(deadline) {
			t.Fatal("the shadow terminal was not restarted")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := v.sync(); err != nil {
		t.Fatal(err)
	}
	if !logged("shadow terminal stopped") || !logged("restarting it in 10ms") {
		t.Errorf("logs = %q", logs)
	}
	got := strings.Join(childFrames(t, log, 8), " ")
	if strings.Count(got, `D"history"`) != 1 || !strings.Contains(got, `D"history more"`) {
		t.Errorf("the new child was not fed the history: %s", got)
	}
	// It cannot start again: each try waits longer, until it is closed.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	_ = syscall.Kill(pid(), syscall.SIGKILL)
	waitUntil(t, func() bool { return logged("trying again in 40ms") })
	if _, _, err := v.Snapshot(80, 24, 0, time.Second); err == nil {
		t.Error("a snapshot worked with no shadow terminal")
	}
	v.Feed(22, []byte("x")) // dropped, without a child
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	v.Feed(23, []byte("y"))
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestVTLogWriter(t *testing.T) {
	var got []string
	w := &vtLogWriter{logf: func(f string, a ...any) { got = append(got, fmt.Sprintf(f, a...)) }}
	_, _ = w.Write([]byte("one\ntw"))
	_, _ = w.Write([]byte("o\n"))
	_, _ = w.Write(bytes.Repeat([]byte("x"), 5000))
	if len(got) != 3 || got[0] != "shadow terminal: one" || got[1] != "shadow terminal: two" || len(got[2]) != 5017 {
		t.Errorf("logged %d lines: %.40q", len(got), got)
	}
}
