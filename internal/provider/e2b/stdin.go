package e2b

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// maxPending is how much stdin may wait to be sent before Write blocks.
const maxPending = 1 << 20

// sendWriter carries stdin in SendInput calls, one at a time and in order,
// as the E2B SDKs and CLI do. Write only queues the bytes, so keys typed
// while a call is in flight go together in the next one.
type sendWriter struct {
	p *proc

	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
	err    error
	done   chan struct{}
}

func newSendWriter(p *proc) *sendWriter {
	w := &sendWriter{p: p, done: make(chan struct{})}
	w.cond = sync.NewCond(&w.mu)
	go w.loop()
	return w
}

func (w *sendWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for len(w.buf) > maxPending && w.err == nil && !w.closed {
		w.cond.Wait()
	}
	if w.err != nil {
		return 0, w.err
	}
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	w.buf = append(w.buf, b...)
	w.cond.Broadcast()
	return len(b), nil
}

// next waits for bytes to send. It returns nil when stdin is closed and
// everything is sent, or when sending failed.
func (w *sendWriter) next() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	for len(w.buf) == 0 && !w.closed && w.err == nil {
		w.cond.Wait()
	}
	if w.err != nil || len(w.buf) == 0 {
		return nil
	}
	n := min(len(w.buf), chunk)
	out := append([]byte(nil), w.buf[:n]...)
	w.buf = append(w.buf[:0], w.buf[n:]...)
	w.cond.Broadcast()
	return out
}

func (w *sendWriter) loop() {
	defer close(w.done)
	for {
		b := w.next()
		if b == nil {
			break
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := w.p.e.unary(ctx, "SendInput", sendInputRequest{Process: selector{Pid: w.p.pid}, Input: processInput{Stdin: b}})
		cancel()
		if err != nil {
			w.fail(err)
			return
		}
	}
	w.mu.Lock()
	failed := w.err != nil
	w.mu.Unlock()
	if !failed {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = w.p.e.unary(ctx, "CloseStdin", map[string]selector{"process": {Pid: w.p.pid}})
	}
}

func (w *sendWriter) fail(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err == nil {
		w.err = err
	}
	w.cond.Broadcast()
}

// Close sends what is queued, then EOF.
func (w *sendWriter) Close() error {
	w.mu.Lock()
	w.closed = true
	w.cond.Broadcast()
	w.mu.Unlock()
	select {
	case <-w.done:
	case <-time.After(10 * time.Second):
		w.fail(errors.New("stdin was not sent in time"))
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}

// streamWriter carries stdin in one StreamInput request: one envelope for
// each write, in order, over a single long request.
type streamWriter struct {
	p    *proc
	pw   *io.PipeWriter
	mu   sync.Mutex
	err  error
	done chan struct{}
}

func newStreamWriter(p *proc) *streamWriter {
	pr, pw := io.Pipe()
	w := &streamWriter{p: p, pw: pw, done: make(chan struct{})}
	go w.run(pr)
	first := streamInputRequest{Start: &struct {
		Process selector `json:"process"`
	}{Process: selector{Pid: p.pid}}}
	if err := w.send(first); err != nil {
		w.setErr(err)
	}
	return w
}

func (w *streamWriter) run(pr *io.PipeReader) {
	defer close(w.done)
	req, err := w.p.e.request(context.Background(), "StreamInput", pr, true)
	if err != nil {
		w.setErr(err)
		_ = pr.CloseWithError(err)
		return
	}
	resp, err := w.p.e.client().Do(req)
	if err != nil {
		w.setErr(fmt.Errorf("envd StreamInput: %w", err))
		_ = pr.CloseWithError(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("envd StreamInput: %w", responseError(resp))
		w.setErr(err)
		_ = pr.CloseWithError(err)
		return
	}
	for {
		flags, msg, err := readEnvelope(resp.Body)
		if err != nil {
			return
		}
		if flags&flagEndStream != 0 {
			if err := endError(msg); err != nil {
				w.setErr(err)
			}
			return
		}
	}
}

func (w *streamWriter) setErr(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err == nil {
		w.err = err
	}
}

func (w *streamWriter) send(m streamInputRequest) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := writeEnvelope(&buf, 0, b); err != nil {
		return err
	}
	_, err = w.pw.Write(buf.Bytes())
	return err
}

func (w *streamWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	err := w.err
	w.mu.Unlock()
	if err != nil {
		return 0, err
	}
	for off := 0; off < len(b); off += chunk {
		end := min(off+chunk, len(b))
		m := streamInputRequest{Data: &struct {
			Input processInput `json:"input"`
		}{Input: processInput{Stdin: b[off:end]}}}
		if err := w.send(m); err != nil {
			return off, err
		}
	}
	return len(b), nil
}

// Close ends the input stream, then sends EOF to the command.
func (w *streamWriter) Close() error {
	_ = w.pw.Close()
	select {
	case <-w.done:
	case <-time.After(10 * time.Second):
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := w.p.e.unary(ctx, "CloseStdin", map[string]selector{"process": {Pid: w.p.pid}}); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}
