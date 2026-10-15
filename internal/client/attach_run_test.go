package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/host/hosttest"
)

// The tests in this binary may start it again as a session's holder.
func TestMain(m *testing.M) { hosttest.Main(m) }

// rawReady is what the client writes once the terminal is in raw mode: it
// saves the window title. Keys typed before it would be cooked.
const rawReady = "\x1b[22;0t"

type attachRun struct {
	t     *testing.T
	h     *hosttest.Host
	ch    client.Host
	tr    *hosttest.Transport
	term  *hosttest.Terminal
	state client.AttachState
}

func newAttachRun(t *testing.T) *attachRun {
	t.Helper()
	h := hosttest.New(t)
	ch, tr := h.ClientHost("box")
	return &attachRun{t: t, h: h, ch: ch, tr: tr, term: hosttest.NewTerminal(t, 80, 24)}
}

// session starts a session that runs a shell script.
func (r *attachRun) session(script string) string {
	r.t.Helper()
	s, err := host.New(host.NewArgs{Name: "s", Cwd: "~", Argv: []string{"/bin/sh", "-c", script}, Cols: 80, Rows: 24})
	if err != nil {
		r.t.Fatal(err)
	}
	return s.ID
}

type attachEnd struct {
	res client.AttachResult
	err error
}

// attach runs client.Attach in the background.
func (r *attachRun) attach(ctx context.Context, id string, settings client.Settings) chan attachEnd {
	r.t.Helper()
	done := make(chan attachEnd, 1)
	go func() {
		res, err := client.Attach(ctx, client.AttachOptions{Host: r.ch, Session: id, Settings: settings, State: &r.state, In: r.term.TTY, Out: r.term.TTY})
		done <- attachEnd{res, err}
	}()
	return done
}

func (r *attachRun) end(done chan attachEnd) attachEnd {
	r.t.Helper()
	select {
	case e := <-done:
		return e
	case <-time.After(20 * time.Second):
		r.t.Fatalf("the attach did not end; the terminal shows %q", r.term.Output())
		return attachEnd{}
	}
}

// Each attach stops reading the terminal before it returns, so the next one
// (or the launcher) gets every key, and nothing reads a file it has closed.
// On Linux a reader left behind took the next attach's keys, and the race
// detector saw it close.
func TestAttachAgainAndAgain(t *testing.T) {
	r := newAttachRun(t)
	id := r.session("stty raw -echo; exec cat")
	for i := range 30 {
		ctx, cancel := context.WithCancel(context.Background())
		done := r.attach(ctx, id, client.DefaultSettings())
		r.term.WaitFor(rawReady)
		r.term.Clear()
		// The agent echoes what it gets: no earlier attach took the keys.
		mark := fmt.Sprintf("MARK%d.", i)
		r.term.Type(mark)
		r.term.WaitFor(mark)
		cancel()
		if e := r.end(done); e.err != nil || !e.res.Detached {
			t.Fatalf("attach %d = %+v, %v", i, e.res, e.err)
		}
	}
}
