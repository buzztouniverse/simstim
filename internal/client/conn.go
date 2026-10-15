package client

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/proto"
	"github.com/iAmLakshya/simstim/internal/version"
)

// ErrClosed is returned after the connection to the host is gone.
var ErrClosed = errors.New("the connection to the host closed")

// Conn is one bridge connection to a host: an `ssh -T` running the helper's
// bridge, carrying framed requests, an attach stream and uploads.
type Conn struct {
	Host     Host
	Features []string

	cmd    *exec.Cmd
	w      *proto.Writer
	stdin  io.WriteCloser
	nextID atomic.Uint64

	mu      sync.Mutex
	pending map[uint64]chan proto.Response
	stream  chan proto.Frame // attach frames: Attached, Output, Ended, Error
	err     error

	lastRecv atomic.Int64
	done     chan struct{}
	stderr   strings.Builder
}

// ErrNotInstalled means the host has no helper of this client's version.
var ErrNotInstalled = errors.New("the helper is not installed")

// Dial starts the bridge on the host and does the hello. For a saved host (one
// the user added, and so agreed to the helper on), a missing helper of this
// version is installed first: an upgrade never needs the user.
func Dial(ctx context.Context, h Host, interactive bool) (*Conn, error) {
	c, err := dial(ctx, h, interactive)
	if !errors.Is(err, ErrNotInstalled) {
		return c, err
	}
	if _, ferr := FindHost(h.Name); ferr != nil {
		return nil, &UserError{Msg: fmt.Sprintf("the %s helper is not installed on %s", brand.Name, h.Name), Hint: brand.Name + " host add " + h.Name}
	}
	s := &SSH{Host: h, Interactive: interactive}
	p, perr := Probe(ctx, s)
	if perr != nil {
		return nil, perr
	}
	if !p.Installed {
		if ierr := Install(ctx, s, p); ierr != nil {
			return nil, ierr
		}
	}
	return dial(ctx, h, interactive)
}

func dial(ctx context.Context, h Host, interactive bool) (*Conn, error) {
	s := &SSH{Host: h, Interactive: interactive}
	cmd, err := s.Command(context.Background(), RemoteHelper(HelperVersion())+" bridge", "-T")
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	c := &Conn{Host: h, cmd: cmd, stdin: stdin, w: proto.NewWriter(stdin),
		pending: map[uint64]chan proto.Response{}, stream: make(chan proto.Frame, 4096), done: make(chan struct{})}
	cmd.Stderr = &lockedWriter{b: &c.stderr}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	r := bufio.NewReaderSize(stdout, 256<<10)
	if err := c.w.WriteJSON(proto.THello, proto.Hello{Magic: proto.Magic, Protocol: version.Protocol, Version: version.Version, Role: "client"}); err != nil {
		c.kill()
		return nil, c.startError(err)
	}
	type res struct {
		f   proto.Frame
		err error
	}
	first := make(chan res, 1)
	go func() { f, err := proto.Read(r); first <- res{f, err} }()
	select {
	case <-ctx.Done():
		c.kill()
		return nil, ctx.Err()
	case x := <-first:
		if x.err != nil {
			c.kill()
			return nil, c.startError(x.err)
		}
		if x.f.Type != proto.THello {
			c.kill()
			return nil, fmt.Errorf("unexpected first frame from host")
		}
		var hello proto.Hello
		if err := json.Unmarshal(x.f.Payload, &hello); err != nil {
			c.kill()
			return nil, err
		}
		if err := hello.Check(version.Protocol); err != nil {
			c.kill()
			return nil, &UserError{Msg: err.Error(), Hint: brand.Name + " host add " + h.Name + "   (reinstalls the helper)"}
		}
		c.Features = hello.Features
	}
	c.lastRecv.Store(time.Now().UnixNano())
	go c.readLoop(r)
	go func() {
		_ = cmd.Wait()
		c.fail(ErrClosed)
	}()
	return c, nil
}

type lockedWriter struct {
	mu sync.Mutex
	b  *strings.Builder
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.b.Len() < 8192 {
		l.b.Write(p)
	}
	return len(p), nil
}

func (c *Conn) startError(err error) error {
	_ = c.cmd.Wait()
	msg := strings.TrimSpace(c.stderr.String())
	if strings.Contains(msg, "No such file") || strings.Contains(msg, "not found") {
		return ErrNotInstalled
	}
	if msg != "" {
		return sshError(c.Host, &exec.ExitError{}, msg)
	}
	return err
}

func (c *Conn) readLoop(r *bufio.Reader) {
	for {
		f, err := proto.Read(r)
		if err != nil {
			c.fail(ErrClosed)
			return
		}
		c.lastRecv.Store(time.Now().UnixNano())
		switch f.Type {
		case proto.TResponse:
			var resp proto.Response
			if json.Unmarshal(f.Payload, &resp) == nil {
				c.mu.Lock()
				ch := c.pending[resp.ID]
				delete(c.pending, resp.ID)
				c.mu.Unlock()
				if ch != nil {
					ch <- resp
				}
			}
		case proto.TPong:
		case proto.TAttached, proto.TOutput, proto.TEnded, proto.TError:
			select {
			case c.stream <- f:
			case <-c.done:
				return
			}
		}
	}
}

func (c *Conn) fail(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return
	}
	c.err = err
	close(c.done)
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
}

func (c *Conn) kill() {
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

// Close ends the connection.
func (c *Conn) Close() {
	_ = c.stdin.Close()
	select {
	case <-c.done:
	case <-time.After(500 * time.Millisecond):
		c.kill()
	}
	c.fail(ErrClosed)
}

// Done is closed when the connection ends.
func (c *Conn) Done() <-chan struct{} { return c.done }

// LastRecv is when the last frame arrived.
func (c *Conn) LastRecv() time.Time { return time.Unix(0, c.lastRecv.Load()) }

// Ping sends a heartbeat.
func (c *Conn) Ping() error { return c.w.Write(proto.TPing, nil) }

// Call sends a request and decodes the result into out.
func (c *Conn) Call(ctx context.Context, op string, args any, out any) error {
	id := c.nextID.Add(1)
	req := proto.Request{ID: id, Op: op}
	if args != nil {
		b, err := json.Marshal(args)
		if err != nil {
			return err
		}
		req.Args = b
	}
	ch := make(chan proto.Response, 1)
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return c.err
	}
	c.pending[id] = ch
	c.mu.Unlock()
	if err := c.w.WriteJSON(proto.TRequest, req); err != nil {
		return ErrClosed
	}
	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return ctx.Err()
	case resp, ok := <-ch:
		if !ok {
			return ErrClosed
		}
		if !resp.OK {
			if resp.Error != nil {
				return &UserError{Msg: resp.Error.Message, Hint: resp.Error.Hint}
			}
			return errors.New("the host refused the request")
		}
		if out != nil && len(resp.Result) > 0 {
			return json.Unmarshal(resp.Result, out)
		}
		return nil
	}
}

// Stream returns attach frames.
func (c *Conn) Stream() <-chan proto.Frame { return c.stream }

// Send writes a raw frame.
func (c *Conn) Send(t proto.Type, p []byte) error { return c.w.Write(t, p) }

// SendJSON writes a JSON frame.
func (c *Conn) SendJSON(t proto.Type, v any) error { return c.w.WriteJSON(t, v) }

// Upload sends a file into a session's paste folder and returns its path on the host.
func (c *Conn) Upload(ctx context.Context, session, name string, data []byte) (string, error) {
	id := c.nextID.Add(1)
	ch := make(chan proto.Response, 1)
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return "", c.err
	}
	c.pending[id] = ch
	c.mu.Unlock()
	if err := c.w.WriteJSON(proto.TUploadBegin, proto.UploadBegin{ID: id, Session: session, Name: name, Size: int64(len(data))}); err != nil {
		return "", ErrClosed
	}
	for off := 0; off < len(data); off += 256 << 10 {
		end := off + 256<<10
		if end > len(data) {
			end = len(data)
		}
		if err := c.w.Write(proto.TUploadChunk, proto.EncodeChunk(id, data[off:end])); err != nil {
			return "", ErrClosed
		}
	}
	sum := sha256.Sum256(data)
	if err := c.w.WriteJSON(proto.TUploadEnd, proto.UploadEnd{ID: id, SHA256: hex.EncodeToString(sum[:])}); err != nil {
		return "", ErrClosed
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case resp, ok := <-ch:
		if !ok {
			return "", ErrClosed
		}
		if !resp.OK {
			if resp.Error != nil {
				return "", &UserError{Msg: resp.Error.Message, Hint: resp.Error.Hint}
			}
			return "", errors.New("upload failed")
		}
		var r proto.UploadResult
		if err := json.Unmarshal(resp.Result, &r); err != nil {
			return "", err
		}
		return r.Path, nil
	}
}
