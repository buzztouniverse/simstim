package host

import (
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iamlakshya/simstim/internal/brand"
)

func TestIdleDecide(t *testing.T) {
	const after = 30 * time.Minute
	t0 := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	boot := "boot-1"
	at := func(m int) time.Time { return t0.Add(time.Duration(m) * time.Minute) }

	// The first check starts counting, and never powers off.
	st, off := IdleDecide(at(0), IdleState{}, boot, false, after)
	if off || !st.BusyAt.Equal(at(0)) || st.BootID != boot {
		t.Fatalf("first check: %+v, %v", st, off)
	}
	// Idle for less than the idle time: keep counting from the same start.
	st, off = IdleDecide(at(29), st, boot, false, after)
	if off || !st.BusyAt.Equal(at(0)) {
		t.Fatalf("at 29 min: %+v, %v", st, off)
	}
	// A client or a working session resets the count.
	st, off = IdleDecide(at(29), st, boot, true, after)
	if off || !st.BusyAt.Equal(at(29)) {
		t.Fatalf("busy: %+v, %v", st, off)
	}
	if _, off = IdleDecide(at(58), st, boot, false, after); off {
		t.Fatal("powered off 29 minutes after the last activity")
	}
	if _, off = IdleDecide(at(59), st, boot, false, after); !off {
		t.Fatal("did not power off 30 minutes after the last activity")
	}
	// A new boot starts again, whatever the old state says: the VM was
	// woken for a reason.
	st2, off := IdleDecide(at(500), st, "boot-2", false, after)
	if off || st2.BootID != "boot-2" || !st2.BusyAt.Equal(at(500)) {
		t.Fatalf("new boot: %+v, %v", st2, off)
	}
	// A clock that went backwards starts again too.
	if st3, off := IdleDecide(at(10), st, boot, false, after); off || !st3.BusyAt.Equal(at(10)) {
		t.Fatalf("clock back: %+v, %v", st3, off)
	}
}

func TestIdleInterval(t *testing.T) {
	for after, want := range map[time.Duration]time.Duration{
		2 * time.Minute: time.Minute, 10 * time.Minute: 2 * time.Minute, 30 * time.Minute: 5 * time.Minute, 4 * time.Hour: 5 * time.Minute,
	} {
		if got := IdleInterval(after); got != want {
			t.Errorf("IdleInterval(%v) = %v, want %v", after, got, want)
		}
	}
}

func TestIdleUnits(t *testing.T) {
	service, timer := IdleUnits("/h/bin/current/helper", 30*time.Minute)
	if !strings.Contains(service, "ExecStart=/h/bin/current/helper idle check --after 30m0s") || !strings.Contains(service, "Type=oneshot") {
		t.Fatalf("service:\n%s", service)
	}
	if !strings.Contains(timer, "OnUnitActiveSec=5m0s") || !strings.Contains(timer, "WantedBy=timers.target") {
		t.Fatalf("timer:\n%s", timer)
	}
	if !strings.Contains(service, brand.DisplayName) {
		t.Fatal("the unit should say whose it is")
	}
}

func TestIsClientCmdline(t *testing.T) {
	bin := "/home/u/.local/share/x/bin/v1/" + brand.HostBinary
	for cmdline, want := range map[string]bool{
		bin + "\x00bridge\x00":                      true,
		bin + "\x00bridge\x00--idle-exit\x0030\x00": true,
		bin + "\x00attach\x00abc\x00":               true,
		bin + "\x00holder\x00abc\x00":               false,
		"sshd: u@notty\x00":                         false,
		"/bin/sh\x00-c\x00" + bin + " bridge\x00":   false,
	} {
		if got := isClientCmdline([]byte(cmdline)); got != want {
			t.Errorf("%q: got %v", cmdline, got)
		}
	}
}

func TestWatchInput(t *testing.T) {
	pr, pw := io.Pipe()
	var fired atomic.Bool
	r := WatchInput(pr, 300*time.Millisecond, func() { fired.Store(true) })
	go func() { _, _ = io.Copy(io.Discard, r) }()
	// Pings keep it alive.
	for range 6 {
		time.Sleep(100 * time.Millisecond)
		_, _ = pw.Write([]byte("ping"))
	}
	if fired.Load() {
		t.Fatal("fired while input was arriving")
	}
	// Silence ends it.
	deadline := time.Now().Add(3 * time.Second)
	for !fired.Load() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !fired.Load() {
		t.Fatal("did not fire after the input stopped")
	}
	_ = pw.Close()
}

func TestBridgeInputWithoutFlag(t *testing.T) {
	in := strings.NewReader("x")
	if BridgeInput(in, nil) != io.Reader(in) || BridgeInput(in, []string{"--idle-exit", "nope"}) != io.Reader(in) {
		t.Fatal("without a valid --idle-exit the input must be used as it is")
	}
}
