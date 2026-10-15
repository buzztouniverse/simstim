package client

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/iAmLakshya/simstim/internal/api"
)

// Session is a session row, as the host reports it.
type Session = api.SessionInfo

// HostInfo is the host doctor's report.
type HostInfo = api.HostInfo

// NewArgs describes a new session.
type NewArgs = api.NewArgs

// TerminalArgs fills the terminal identity of a new session from this terminal.
func TerminalArgs(a NewArgs, cols, rows int) NewArgs {
	a.Term = os.Getenv("TERM")
	a.ColorTerm = os.Getenv("COLORTERM")
	a.TermProgram = os.Getenv("TERM_PROGRAM")
	a.TermProgramVersion = os.Getenv("TERM_PROGRAM_VERSION")
	a.Cols, a.Rows = cols, rows
	return a
}

// Sessions lists a host's sessions.
func (c *Conn) Sessions(ctx context.Context) ([]Session, error) {
	var out []Session
	err := c.Call(ctx, "ls", nil, &out)
	return out, err
}

// NewSession starts a session.
func (c *Conn) NewSession(ctx context.Context, a NewArgs) (*Session, error) {
	var s Session
	if err := c.Call(ctx, "new", a, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteSession stops and removes a session.
func (c *Conn) DeleteSession(ctx context.Context, id string) error {
	return c.Call(ctx, "delete", map[string]string{"id": id}, nil)
}

// RenameSession renames a session.
func (c *Conn) RenameSession(ctx context.Context, id, name string) error {
	return c.Call(ctx, "rename", map[string]string{"id": id, "name": name}, nil)
}

// Doctor asks the host to check itself.
func (c *Conn) Doctor(ctx context.Context) (HostInfo, error) {
	var d HostInfo
	err := c.Call(ctx, "doctor", nil, &d)
	return d, err
}

// Peek returns the newest output of a session, for a preview.
func (c *Conn) Peek(ctx context.Context, id string, n int) (data []byte, cols, rows int, err error) {
	var r struct {
		Data []byte `json:"data"`
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
	}
	err = c.Call(ctx, "peek", map[string]any{"id": id, "bytes": n}, &r)
	return r.Data, r.Cols, r.Rows, err
}

// SetupResult reports what host setup did.
type SetupResult = api.SetupResult

// Setup turns on linger and installs this terminal's terminfo on the host.
func (c *Conn) Setup(ctx context.Context) (SetupResult, error) {
	args := api.SetupArgs{Linger: true, Terminfo: map[string]string{}}
	if t := os.Getenv("TERM"); t != "" && t != "xterm-256color" && t != "xterm" {
		if src, err := exec.Command("infocmp", "-x", t).Output(); err == nil {
			args.Terminfo[t] = string(src)
		}
	}
	var r SetupResult
	err := c.Call(ctx, "setup", args, &r)
	return r, err
}

// Ping measures the round trip through the bridge.
func (c *Conn) RoundTrip(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	var out []Session
	if err := c.Call(ctx, "ls", nil, &out); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

// SendText types text into a session without attaching.
func (c *Conn) SendText(ctx context.Context, id, text string, enter bool) error {
	return c.Call(ctx, "send", map[string]any{"id": id, "text": text, "enter": enter}, nil)
}

// RestoreResult is one session that restore looked at.
type RestoreResult = api.RestoreResult

// Restore asks the host to start again sessions a restart stopped.
func (c *Conn) Restore(ctx context.Context, all bool) ([]RestoreResult, error) {
	var out []RestoreResult
	err := c.Call(ctx, "restore", map[string]bool{"all": all}, &out)
	return out, err
}
