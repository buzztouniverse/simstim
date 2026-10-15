package tui

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/client/helpers"
)

// Backend is everything the launcher needs from outside: saved hosts,
// SSH, the host helper and attach. The real one wraps package client. Tests
// use a fake, so every screen can be checked without SSH.
type Backend interface {
	LoadHosts() ([]client.Host, error)
	SaveHosts([]client.Host) error
	Settings() client.Settings
	// Dial opens a background connection. It never prompts.
	Dial(ctx context.Context, h client.Host) (Conn, error)
	// Probe, Install and HelperSize serve the add-host wizard.
	Probe(ctx context.Context, h client.Host) (client.Platform, error)
	Install(ctx context.Context, h client.Host, p client.Platform) error
	HelperSize(p client.Platform) (int, bool)
	HelperVersion() string
	// SSHConfigHosts lists the Host aliases in the user's SSH config.
	SSHConfigHosts() []string
	// Attach runs a session in this terminal until the user detaches.
	Attach(ctx context.Context, o client.AttachOptions) (client.AttachResult, error)
	// Memory and Remember read and write the launcher's memory.
	Memory() memory
	Remember(memory)
	// Secrets fetches the values of a host's own secrets for a new
	// session. It never prompts.
	Secrets(ctx context.Context, h client.Host) (map[string]api.Secret, error)
	// SetupTrust and RememberSetup read and keep the user's answers about
	// repositories' setup scripts on a host.
	SetupTrust(host string) map[string]bool
	RememberSetup(host, repo string, run bool) error
}

// Conn is one connection to a host. *client.Conn satisfies it.
type Conn interface {
	Sessions(ctx context.Context) ([]client.Session, error)
	NewSession(ctx context.Context, a client.NewArgs) (*client.Session, error)
	Delete(ctx context.Context, id string) (client.DeleteResult, error)
	RenameSession(ctx context.Context, id, name string) error
	Doctor(ctx context.Context) (client.HostInfo, error)
	Peek(ctx context.Context, id string, n int) ([]byte, int, int, error)
	Setup(ctx context.Context) (client.SetupResult, error)
	RoundTrip(ctx context.Context) (time.Duration, error)
	Folders(ctx context.Context, prefix string) (client.FoldersResult, error)
	Done() <-chan struct{}
	Close()
}

var _ Conn = (*client.Conn)(nil)

// realBackend talks to real hosts through package client.
type realBackend struct{}

func (realBackend) LoadHosts() ([]client.Host, error) { return client.LoadHosts() }
func (realBackend) SaveHosts(h []client.Host) error   { return client.SaveHosts(h) }
func (realBackend) Settings() client.Settings         { return client.LoadSettings() }
func (realBackend) HelperVersion() string             { return client.HelperVersion() }
func (realBackend) Memory() memory                    { return loadMemory() }
func (realBackend) Remember(mem memory)               { _ = saveMemory(mem) }

func (realBackend) SetupTrust(host string) map[string]bool { return client.SetupTrust(host) }
func (realBackend) RememberSetup(host, repo string, run bool) error {
	return client.RememberSetup(host, repo, run)
}

func (realBackend) Secrets(ctx context.Context, h client.Host) (map[string]api.Secret, error) {
	return client.FetchSecrets(ctx, client.LoadSettings(), client.SecretNames(h, nil), false)
}

func (realBackend) Dial(ctx context.Context, h client.Host) (Conn, error) {
	// Opening the launcher must not boot every stopped VM or resume every
	// paused sandbox; it says they are stopped instead.
	c, err := client.Dial(client.WithoutWake(ctx), h, false)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (realBackend) Probe(ctx context.Context, h client.Host) (client.Platform, error) {
	return client.Probe(ctx, &client.SSH{Host: h})
}

func (realBackend) Install(ctx context.Context, h client.Host, p client.Platform) error {
	return client.Install(ctx, &client.SSH{Host: h}, p)
}

func (realBackend) HelperSize(p client.Platform) (int, bool) {
	b, ok := helpers.For(p.OS, p.Arch)
	return len(b), ok
}

func (realBackend) SSHConfigHosts() []string { return client.SSHConfigHosts() }

func (realBackend) Attach(ctx context.Context, o client.AttachOptions) (client.AttachResult, error) {
	return client.Attach(ctx, o)
}

// connSet tracks every open connection, so quitting closes them all, even
// one whose dial finishes after the screen has gone.
type connSet struct {
	mu     sync.Mutex
	conns  map[Conn]struct{}
	closed bool
}

func newConnSet() *connSet { return &connSet{conns: map[Conn]struct{}{}} }

// add records c. It closes c and returns false after closeAll.
func (s *connSet) add(c Conn) bool {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		c.Close()
		return false
	}
	s.conns[c] = struct{}{}
	s.mu.Unlock()
	return true
}

// drop closes c and forgets it.
func (s *connSet) drop(c Conn) {
	s.mu.Lock()
	delete(s.conns, c)
	s.mu.Unlock()
	c.Close()
}

// closeAll closes every connection, in parallel, and refuses new ones.
func (s *connSet) closeAll() {
	s.mu.Lock()
	s.closed = true
	all := make([]Conn, 0, len(s.conns))
	for c := range s.conns {
		all = append(all, c)
	}
	clear(s.conns)
	s.mu.Unlock()
	var wg sync.WaitGroup
	for _, c := range all {
		wg.Go(c.Close)
	}
	wg.Wait()
}

func asUserError(err error) (*client.UserError, bool) {
	var ue *client.UserError
	if errors.As(err, &ue) {
		return ue, true
	}
	return nil, false
}
