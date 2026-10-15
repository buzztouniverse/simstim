package tui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/iAmLakshya/simstim/internal/client"
)

// Demo mode shows the home screen with sample hosts and sessions. Nothing
// connects anywhere. It lets people try the screens before they add a host,
// and it is how the screenshots in the README are made.

// RunDemo opens the home screen on sample data. With empty, it shows the
// first-run screen instead.
func RunDemo(ctx context.Context, empty bool) error {
	return runWith(ctx, newDemoBackend(empty), Options{})
}

type demoBackend struct {
	mu       sync.Mutex
	hosts    []client.Host
	sessions map[string][]client.Session
	offline  map[string]bool
}

func newDemoBackend(empty bool) *demoBackend {
	b := &demoBackend{sessions: map[string][]client.Session{}, offline: map[string]bool{"hetzner-box": true}}
	if empty {
		return b
	}
	now := time.Now()
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	b.hosts = []client.Host{
		{Name: "homelab", SSH: "homelab", Labels: map[string]string{"provider": "proxmox"}},
		{Name: "gcp-dev", SSH: "gcp-dev", Labels: map[string]string{"provider": "gcp"}},
		{Name: "hetzner-box", SSH: "hetzner-box"},
	}
	b.sessions["homelab"] = []client.Session{
		{ID: "k2v7m4qa", Name: "auth-refactor", Agent: "claude", Cwd: "/home/me/src/app", State: "needs_input", Source: "screen:permission_prompt",
			Reason: "Do you want to proceed?", CreatedAt: ago(2 * time.Minute), Since: ago(40 * time.Second), Alive: true,
			MemoryCurrent: 612 << 20, MemoryMax: 8 << 30, Cols: 120, Rows: 34},
		{ID: "p9x3hdte", Name: "api-tests", Agent: "codex", Cwd: "/home/me/src/api", State: "working", Source: "screen:esc_to_interrupt",
			CreatedAt: ago(14 * time.Second), Since: ago(5 * time.Second), Alive: true, MemoryCurrent: 140 << 20, MemoryMax: 8 << 30, Cols: 120, Rows: 34},
		{ID: "w5c8n2rb", Name: "docs-site", Agent: "opencode", Cwd: "/home/me/src/docs", State: "idle", Source: "quiet",
			CreatedAt: ago(time.Hour), Since: ago(20 * time.Minute), Alive: true, MemoryCurrent: 380 << 20, MemoryMax: 8 << 30, Cols: 120, Rows: 34},
		{ID: "j6r4y7uf", Name: "notes", Agent: "shell", Cwd: "/home/me", State: "exited", CreatedAt: ago(3 * time.Hour), Since: ago(2 * time.Hour),
			ExitCode: new(int), Alive: true, Cols: 120, Rows: 34},
	}
	b.sessions["gcp-dev"] = []client.Session{
		{ID: "t3f8a1zk", Name: "data-pipeline", Agent: "gemini", Cwd: "/home/me/pipeline", State: "working", Source: "hook",
			CreatedAt: ago(26 * time.Minute), Since: ago(3 * time.Second), Alive: true, MemoryCurrent: 290 << 20, MemoryMax: 6 << 30, Cols: 120, Rows: 34},
	}
	return b
}

func (b *demoBackend) LoadHosts() ([]client.Host, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]client.Host(nil), b.hosts...), nil
}

func (b *demoBackend) SaveHosts(h []client.Host) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hosts = append([]client.Host(nil), h...)
	return nil
}

func (b *demoBackend) Settings() client.Settings {
	s := client.LoadSettings()
	s.Keymap = nil
	return s
}

func (b *demoBackend) HelperVersion() string { return "0.1.0" }

func (b *demoBackend) Dial(_ context.Context, h client.Host) (Conn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.offline[h.Name] {
		return nil, &client.UserError{Msg: "cannot reach " + h.Name + ": connection timed out", Hint: "ssh " + h.SSH}
	}
	return &demoConn{be: b, host: h.Name, done: make(chan struct{})}, nil
}

func (b *demoBackend) Probe(context.Context, client.Host) (client.Platform, error) {
	return client.Platform{OS: "linux", Arch: "amd64"}, nil
}

func (b *demoBackend) Install(context.Context, client.Host, client.Platform) error {
	time.Sleep(600 * time.Millisecond)
	return nil
}

func (b *demoBackend) HelperSize(client.Platform) (int, bool) { return 6_400_000, true }

func (b *demoBackend) SSHConfigHosts() []string {
	return []string{"homelab", "gcp-dev", "hetzner-box", "work-vm", "pi"}
}

func (b *demoBackend) Attach(context.Context, client.AttachOptions) (client.AttachResult, error) {
	return client.AttachResult{}, &client.UserError{Msg: "attach is not available in the demo", Hint: "Add a real host to attach to a session."}
}

type demoConn struct {
	be   *demoBackend
	host string
	done chan struct{}
	once sync.Once
}

func (c *demoConn) Sessions(context.Context) ([]client.Session, error) {
	c.be.mu.Lock()
	defer c.be.mu.Unlock()
	return append([]client.Session(nil), c.be.sessions[c.host]...), nil
}

func (c *demoConn) NewSession(_ context.Context, a client.NewArgs) (*client.Session, error) {
	c.be.mu.Lock()
	defer c.be.mu.Unlock()
	name := a.Name
	if name == "" {
		parts := strings.Split(strings.TrimRight(a.Cwd, "/"), "/")
		name = parts[len(parts)-1]
	}
	s := client.Session{ID: "n" + time.Now().Format("150405"), Name: name, Agent: a.Agent, Cwd: a.Cwd, State: "working",
		CreatedAt: time.Now(), Since: time.Now(), Alive: true, Cols: a.Cols, Rows: a.Rows}
	c.be.sessions[c.host] = append(c.be.sessions[c.host], s)
	return &s, nil
}

func (c *demoConn) Delete(_ context.Context, id string) (client.DeleteResult, error) {
	c.be.mu.Lock()
	defer c.be.mu.Unlock()
	list := c.be.sessions[c.host]
	for i, s := range list {
		if s.ID == id {
			c.be.sessions[c.host] = append(list[:i:i], list[i+1:]...)
			return client.DeleteResult{Deleted: true}, nil
		}
	}
	return client.DeleteResult{}, errors.New("no such session")
}

func (c *demoConn) RenameSession(_ context.Context, id, name string) error {
	c.be.mu.Lock()
	defer c.be.mu.Unlock()
	for i, s := range c.be.sessions[c.host] {
		if s.ID == id {
			c.be.sessions[c.host][i].Name = name
		}
	}
	return nil
}

func (c *demoConn) Doctor(context.Context) (client.HostInfo, error) {
	return client.HostInfo{Hostname: c.host, OS: "linux", Arch: "amd64", Distro: "Debian GNU/Linux 13 (trixie)", CPUs: 4,
		MemTotal: 16 << 30, DiskFree: 61 << 30, Version: "0.1.0", MemoryMax: 6 << 30,
		Agents: map[string]string{"claude": "/home/me/.local/bin/claude", "codex": "/home/me/.local/bin/codex",
			"opencode": "/home/me/.opencode/bin/opencode", "gemini": "/usr/local/bin/gemini", "shell": "/bin/bash"}}, nil
}

func (c *demoConn) Peek(_ context.Context, id string, _ int) ([]byte, int, int, error) {
	return []byte(demoScreen(id)), 120, 34, nil
}

func (c *demoConn) Setup(context.Context) (client.SetupResult, error) {
	return client.SetupResult{}, nil
}

func (c *demoConn) RoundTrip(context.Context) (time.Duration, error) {
	return map[string]time.Duration{"homelab": 4 * time.Millisecond, "gcp-dev": 38 * time.Millisecond}[c.host], nil
}

func (c *demoConn) Folders(_ context.Context, prefix string) (client.FoldersResult, error) {
	r := client.FoldersResult{Recent: []string{"~/src/app", "~/src/api"}, Repos: []string{"~/src/app", "~/src/api", "~/src/docs"}}
	for _, p := range []string{"~/src/api/", "~/src/app/", "~/src/docs/"} {
		if prefix != "" && strings.HasPrefix(p, prefix) {
			r.Matches = append(r.Matches, p)
		}
	}
	return r, nil
}

func (c *demoConn) Done() <-chan struct{} { return c.done }

func (c *demoConn) Close() { c.once.Do(func() { close(c.done) }) }

// demoScreen is a made-up agent screen for the preview.
func demoScreen(id string) string {
	const (
		dim   = "\x1b[2m"
		bold  = "\x1b[1m"
		green = "\x1b[32m"
		red   = "\x1b[31m"
		cyan  = "\x1b[36m"
		mag   = "\x1b[35m"
		reset = "\x1b[0m"
	)
	switch id {
	case "k2v7m4qa":
		return "\r\n" + bold + "> " + reset + "Move the session store to the new token format, and keep old tokens working.\r\n\r\n" +
			mag + "● " + reset + "I'll read the current store first.\r\n" +
			dim + "  Read internal/auth/store.go (212 lines)\r\n  Read internal/auth/token.go (96 lines)" + reset + "\r\n\r\n" +
			mag + "● " + reset + "Plan: add a v2 encoder, read both formats, migrate on write.\r\n\r\n" +
			mag + "● " + reset + "Update(internal/auth/token.go)\r\n" +
			green + "  + func encodeV2(t Token) ([]byte, error) {" + reset + "\r\n" +
			green + "  +     return json.Marshal(tokenV2{ID: t.ID, Exp: t.Exp})" + reset + "\r\n" +
			red + "  - return legacyEncode(t)" + reset + "\r\n\r\n" +
			bold + " Run the migration tests?" + reset + "\r\n" +
			"  go test ./internal/auth/... -run Migrate\r\n\r\n" +
			" Do you want to proceed?\r\n" + cyan + " ❯ 1. Yes" + reset + "\r\n   2. No, and tell the agent what to do differently\r\n"
	case "p9x3hdte":
		return "\r\n" + bold + "> " + reset + "Make the API tests pass on the new schema.\r\n\r\n" +
			dim + "  ran go test ./... (3 failed)" + reset + "\r\n" +
			"  fixing handlers/users.go: the nullable email column\r\n\r\n" +
			cyan + "• Working (14s • esc to interrupt)" + reset + "\r\n"
	default:
		return "\r\n" + dim + "  (idle)" + reset + "\r\n"
	}
}
