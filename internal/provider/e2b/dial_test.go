//go:build !windows

// The host helper, whose bridge this test runs, builds only for Unix.

package e2b

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/provider"
)

// The whole path: a paused sandbox is resumed on attach, the helper's real
// bridge answers over envd's process API, and the client keeps the lease.
func TestDialResumesAndTalksToTheRealBridge(t *testing.T) {
	p, api, envd := testProvider(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	envd.mu.Lock()
	envd.bridge = func(in io.Reader, out io.Writer, _ string) error {
		return host.RunBridge(host.WatchInput(in, time.Hour, func() {}), out)
	}
	envd.mu.Unlock()
	client.RegisterProvider("e2b-test", p.Open)

	h, err := p.Create(context.Background(), provider.Request{Name: "cloud"})
	if err != nil {
		t.Fatal(err)
	}
	h.Machine.Provider = "e2b-test"
	lc, _ := p.Open(h)
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c, err := client.Dial(ctx, h, false)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Sessions(ctx); err != nil {
		t.Fatalf("the bridge does not answer over envd: %v", err)
	}
	envd.mu.Lock()
	bridges := append([]string(nil), envd.bridges...)
	envd.mu.Unlock()
	if len(bridges) != 1 || !strings.HasSuffix(bridges[0], " bridge --idle-exit 30") {
		t.Fatalf("bridge commands = %q", bridges)
	}
	var resumed, leased bool
	deadline := time.Now().Add(5 * time.Second)
	for !leased && time.Now().Before(deadline) {
		for _, call := range api.Calls() {
			resumed = resumed || strings.HasSuffix(call.Path, "/connect")
			leased = leased || (strings.HasSuffix(call.Path, "/timeout") && call.Body["timeout"] == float64(15*60))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !resumed || !leased {
		t.Fatalf("resumed %v, leased for 15m %v; calls %+v", resumed, leased, api.Calls())
	}
}
