package host_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/host/hosttest"
	"github.com/iAmLakshya/simstim/internal/proto"
	"github.com/iAmLakshya/simstim/internal/version"
)

// start starts a session that runs argv in the home folder.
func start(t *testing.T, name string, argv ...string) *host.SessionInfo {
	t.Helper()
	s, err := host.New(host.NewArgs{Name: name, Cwd: "~", Argv: argv, Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("new %s: %v", name, err)
	}
	return s
}

// peek returns the newest output of a session.
func peek(t *testing.T, id string) string {
	t.Helper()
	raw, err := host.HolderRequest(id, "peek", map[string]int{"bytes": 64 << 10}, 3*time.Second)
	if err != nil {
		t.Fatalf("peek: %v", err)
	}
	var r struct {
		Data []byte `json:"data"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return string(r.Data)
}

// waitOutput waits until a session's output holds want.
func waitOutput(t *testing.T, id, want string) {
	t.Helper()
	hosttest.WaitFor(t, 10*time.Second, "output "+want, func() bool { return strings.Contains(peek(t, id), want) })
}

// live returns the holder's own copy of a session's manifest, which is
// current: the file is saved only once a second.
func live(t *testing.T, id string) *host.Manifest {
	t.Helper()
	raw, err := host.HolderRequest(id, "info", nil, 3*time.Second)
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	var m host.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return &m
}

// waitState waits until a session's manifest has the state.
func waitState(t *testing.T, id, state string) *host.Manifest {
	t.Helper()
	var m *host.Manifest
	hosttest.WaitFor(t, 10*time.Second, "state "+state, func() bool {
		var err error
		m, err = host.LoadManifest(id)
		return err == nil && m.Status.State == state
	})
	return m
}

// rawBridge is a client that speaks the bridge's frames itself, for what
// client.Conn hides.
type rawBridge struct {
	t      *testing.T
	w      *proto.Writer
	frames chan proto.Frame
	in     *io.PipeWriter
	done   chan error
}

func newRawBridge(t *testing.T) *rawBridge {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	b := &rawBridge{t: t, w: proto.NewWriter(inW), frames: make(chan proto.Frame, 1024), in: inW, done: make(chan error, 1)}
	go func() {
		err := host.RunBridge(inR, outW)
		_ = outW.CloseWithError(io.EOF)
		b.done <- err
	}()
	go func() {
		r := bufio.NewReader(outR)
		for {
			f, err := proto.Read(r)
			if err != nil {
				close(b.frames)
				return
			}
			b.frames <- f
		}
	}()
	t.Cleanup(func() {
		_ = inW.Close()
		select {
		case <-b.done:
		case <-time.After(5 * time.Second):
			t.Error("the bridge did not end when its input closed")
		}
	})
	if err := b.w.WriteJSON(proto.THello, proto.Hello{Magic: proto.Magic, Protocol: version.Protocol, Role: "client"}); err != nil {
		t.Fatal(err)
	}
	if f := b.next(proto.THello); !bytes.Contains(f.Payload, []byte(`"bridge"`)) {
		t.Fatalf("hello = %s", f.Payload)
	}
	return b
}

// next returns the next frame of one of the types, skipping others.
func (b *rawBridge) next(types ...proto.Type) proto.Frame {
	b.t.Helper()
	timeout := time.After(10 * time.Second)
	for {
		select {
		case f, ok := <-b.frames:
			if !ok {
				b.t.Fatalf("the bridge closed while waiting for %v", types)
			}
			for _, ty := range types {
				if f.Type == ty {
					return f
				}
			}
		case <-timeout:
			b.t.Fatalf("no frame of %v in time", types)
		}
	}
}

func (b *rawBridge) send(ty proto.Type, p []byte) {
	b.t.Helper()
	if err := b.w.Write(ty, p); err != nil {
		b.t.Fatal(err)
	}
}

func (b *rawBridge) sendJSON(ty proto.Type, v any) {
	b.t.Helper()
	if err := b.w.WriteJSON(ty, v); err != nil {
		b.t.Fatal(err)
	}
}

// attach attaches and returns the Attached frame.
func (b *rawBridge) attach(id string, last int64, cols, rows int) proto.Attached {
	b.t.Helper()
	b.sendJSON(proto.TAttach, proto.Attach{Session: id, Cols: cols, Rows: rows, LastOffset: last})
	f := b.next(proto.TAttached, proto.TError)
	if f.Type == proto.TError {
		b.t.Fatalf("attach failed: %s", f.Payload)
	}
	var a proto.Attached
	if err := json.Unmarshal(f.Payload, &a); err != nil {
		b.t.Fatal(err)
	}
	return a
}

// writeFile writes a file, making its folder.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
