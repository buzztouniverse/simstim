package tui

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/onboard"
)

// The demo's guided setup: every place looks ready, and nothing is made,
// checked or changed on this computer.

func (b *demoBackend) Local() onboard.Local { return demoLocal{} }

type demoLocal struct{}

func (demoLocal) Places() []onboard.Place {
	return onboard.Places(onboard.Probe{
		GOOS:      "darwin",
		Has:       func(n string) bool { return n != "limactl" },
		SSHHosts:  []string{"homelab", "gcp-dev", "hetzner-box", "work-vm", "pi"},
		Providers: []string{onboard.Lima, onboard.Docker, onboard.E2B},
	})
}

func (demoLocal) PublicKey() string       { return "~/.ssh/id_ed25519.pub" }
func (demoLocal) HasSSHHost(string) bool  { return false }
func (demoLocal) AddSSHHost(string) error { return nil }
func (demoLocal) SSHWorks(context.Context, string) error {
	time.Sleep(400 * time.Millisecond)
	return nil
}
func (demoLocal) SetIdle(context.Context, client.Host) (string, error) {
	return "powers off after 30m idle", nil
}

func (demoLocal) Create(ctx context.Context, kind, name string, out io.Writer) (client.Host, error) {
	for _, l := range []string{"Starting the container", "Adding the user", "Done"} {
		fmt.Fprintf(out, "  %s\n", l)
		select {
		case <-ctx.Done():
			return client.Host{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return client.Host{Name: name, SSH: name}, nil
}

// AgentChecks says the demo host has Claude Code, logged in, and Codex,
// not logged in.
func (c *demoConn) AgentChecks(context.Context) ([]client.AgentCheck, error) {
	return []client.AgentCheck{
		{Kind: "claude", Title: "Claude Code", Support: "first-class", Installed: true, OnPath: true, Path: "/home/me/.local/bin/claude", Login: api.LoginYes, LoginHow: "`claude auth status` says so"},
		{Kind: "codex", Title: "Codex", Support: "first-class", Installed: true, OnPath: true, Path: "/home/me/.local/bin/codex", Login: api.LoginNo, LoginHow: "`codex login status` says not"},
		{Kind: "opencode", Title: "opencode", Support: "supported"},
		{Kind: "antigravity", Title: "Antigravity CLI", Support: "supported"},
	}, nil
}
