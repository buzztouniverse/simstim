package host

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// This file runs the shadow terminal: a child process, built from vt/, that
// keeps a copy of the session's screen with libghostty-vt. A new client gets
// a snapshot from it instead of a replay that may start mid-stream. The
// child is a separate process, so a crash there cannot take the agent down;
// the VT restarts it and feeds it the session's recent output again.
//
// The frame format is in vt/README.md: a type byte, a little-endian uint32
// payload length, then the payload.

// Shadow terminal frame types.
const (
	// Holder to shadow terminal.
	vtFrameData     = 'D'
	vtFrameResize   = 'R'
	vtFrameSnapshot = 'S'
	vtFrameQueries  = 'Q'
	vtFrameProfile  = 'P'
	vtFrameDump     = 'T'
	vtFrameSync     = 'Y'
	// Shadow terminal to holder. S, T and Y reply with their own type.
	vtFrameWrite = 'W'
	vtFrameEvent = 'E'
)

const (
	// vtMaxBehind is how many bytes may wait for the child before it is
	// dropped and rebuilt from the history. It is twice the ring.
	vtMaxBehind = 2 * RingSize
	// vtChunk is the largest data frame sent to the child.
	vtChunk = 256 << 10
	// vtStableAfter is how long a child must live for its restart delay
	// to start again from the shortest.
	vtStableAfter = time.Minute
)

// vtBackoff is the wait before each restart after a crash, in order. The
// last one repeats, so a child that always crashes on the same output tries
// again only now and then, until that output leaves the ring.
var vtBackoff = []time.Duration{
	200 * time.Millisecond, time.Second, 5 * time.Second, 30 * time.Second, 5 * time.Minute,
}

// ErrVTDown is returned while the shadow terminal is not running, for
// example between a crash and its restart.
var ErrVTDown = errors.New("the shadow terminal is not running. Try: attach again in a moment")

// VTOptions configure a VT.
type VTOptions struct {
	// Path is the shadow terminal executable.
	Path string
	// Cols and Rows are the starting size. Zero means 80 by 24.
	Cols, Rows int
	// Scrollback is how many lines the shadow terminal keeps. Zero means
	// its default, 10,000.
	Scrollback int
	// History returns the session's recent output, and the stream offset of
	// its first byte. Each new child is fed it, at start and after a crash.
	// It is called with no VT lock held. It should take the lock that
	// serialises the calls to Feed, so that each byte is fed exactly once.
	// It may be nil.
	History func() (data []byte, start int64)
	// OnWrite gets bytes to write to the PTY: replies to the agent's
	// terminal queries, sent only while AnswerQueries is on.
	OnWrite func(p []byte)
	// OnEvent gets the events the session's output raises. Events in the
	// history replayed after a crash are not raised again.
	//
	// OnWrite and OnEvent run one at a time on the VT's own goroutine. They
	// must not block for long, or the child waits for them. Both may be nil.
	OnEvent func(e VTEvent)
	// Logf, if set, gets a line for each crash and restart, and what the
	// child writes to stderr.
	Logf func(format string, args ...any)
}

// VTEvent is something the session's output asked the terminal for.
type VTEvent struct {
	// Type is "bell", "notify" (OSC 9 or 777), "title" (OSC 0 or 2),
	// "pwd" (OSC 7), "progress" (OSC 9;4) or "status" (OSC 7501).
	Type string `json:"type"`
	// Title is the notification's title, or the new window title.
	Title string `json:"title,omitempty"`
	// Body is the notification's text.
	Body string `json:"body,omitempty"`
	// State is the progress state (remove, set, error, indeterminate,
	// pause) or the program's status (idle, working, done, blocked, error,
	// clear).
	State string `json:"state,omitempty"`
	// Value is the progress in percent, or -1 if there is none.
	Value int `json:"value"`
	// Data is the pwd URL, or the status report after "7501;".
	Data string `json:"data,omitempty"`
}

// VTProfile is what the shadow terminal says about itself while it answers
// queries. Copy it from the last client's terminal. An empty field keeps
// the default: Ghostty's DA replies, "libghostty" for XTVERSION, and no
// reply to colour or colour-scheme queries.
type VTProfile struct {
	// DA1 is the primary device attributes: the conformance level, then
	// the features, for example 62, 22, 52.
	DA1 []int `json:"da1,omitempty"`
	// DA2 is the secondary device attributes: the device type, the
	// firmware version and the ROM number, for example 1, 10, 0.
	DA2 []int `json:"da2,omitempty"`
	// XTVersion is the name and version, for example "ghostty 1.3.1".
	XTVersion string `json:"xtversion,omitempty"`
	// Foreground, Background and Cursor are the default colours, as
	// "#rrggbb", for OSC 10, 11 and 12 queries.
	Foreground string `json:"fg,omitempty"`
	Background string `json:"bg,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	// ColorScheme is "dark" or "light", for CSI ? 996 n.
	ColorScheme string `json:"color_scheme,omitempty"`
}

// VT runs the shadow terminal and keeps it running. Its methods are safe
// for concurrent use. None of them wait for the child, except Snapshot,
// which waits up to its timeout, and Close, which waits up to a second for
// the child to exit.
type VT struct {
	opt       VTOptions
	maxBehind int

	mu         sync.Mutex
	child      *vtChild
	closed     bool
	done       chan struct{}
	cols, rows int
	xpx, ypx   int
	answering  bool
	profile    []byte // JSON, nil if none was set
	fed        int64  // the stream offset after the last byte fed
	collecting bool   // true while a restart reads the history
	pending    []vtChunkAt
	crashes    int // crashes since the last stable child
}

type vtChunkAt struct {
	off  int64
	data []byte
}

// vtChild is one run of the shadow terminal process.
type vtChild struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	started time.Time
	wake    chan struct{}
	gone    chan struct{} // closed when the process has exited

	// Guarded by VT.mu.
	queue   [][]byte // frames waiting for the writer
	queued  int      // bytes in queue and in the writer's hands
	limit   int
	muted   bool // drop events while the history is replayed
	waiters map[byte][]chan vtReply
	dropped bool // killed for falling behind
	dead    bool
}

type vtReply struct {
	p   []byte
	err error
}

// StartVT starts the shadow terminal and feeds it the history, if there is
// one. It calls History before it returns, so do not hold the lock that
// History takes.
func StartVT(opt VTOptions) (*VT, error) {
	if opt.Cols <= 0 {
		opt.Cols = 80
	}
	if opt.Rows <= 0 {
		opt.Rows = 24
	}
	v := &VT{opt: opt, maxBehind: vtMaxBehind, done: make(chan struct{}), cols: opt.Cols, rows: opt.Rows}
	if err := v.start(); err != nil {
		return nil, err
	}
	return v, nil
}

// Feed gives the shadow terminal the session's output. off is the stream
// offset of p[0]. It never blocks: if the child falls far behind, it is
// dropped and rebuilt from the history.
func (v *VT) Feed(off int64, p []byte) {
	if len(p) == 0 {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return
	}
	v.fed = max(v.fed, off+int64(len(p)))
	if v.collecting {
		// A restart is reading the history. Keep what may come after it.
		v.pending = append(v.pending, vtChunkAt{off, bytes.Clone(p)})
	}
	c := v.child
	if c == nil {
		return
	}
	if c.dropped {
		return
	}
	if c.queued+len(p) > c.limit {
		// It is stuck or too slow. The history will catch up a new one.
		v.logf("shadow terminal is %d bytes behind: restarting it", c.queued)
		c.dropped = true
		c.kill()
		return
	}
	for len(p) > 0 {
		n := min(len(p), vtChunk)
		c.push(vtFrame(vtFrameData, p[:n]))
		p = p[n:]
	}
}

// Resize sets the terminal size: columns and rows, and the window's size in
// pixels if it is known (zero if not).
func (v *VT) Resize(cols, rows, xpx, ypx int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.cols, v.rows, v.xpx, v.ypx = cols, rows, xpx, ypx
	if v.child != nil {
		v.child.push(v.resizeFrame())
	}
}

// Snapshot returns the bytes that redraw the session in a blank terminal of
// cols by rows: up to maxLines lines of scrollback, the screen, the cursor
// and the modes. It also returns the stream offset the snapshot reaches:
// send the client the output from that offset on, and it misses nothing
// and sees nothing twice. It resizes the shadow terminal to cols by rows,
// as the PTY is about to be. It waits at most timeout for the child.
//
// Write it with the cursor at the start of an empty line, with only empty
// lines below; anything above scrolls into the terminal's scrollback.
func (v *VT) Snapshot(cols, rows, maxLines int, timeout time.Duration) (snap []byte, upTo int64, err error) {
	if cols <= 0 || rows <= 0 {
		return nil, 0, fmt.Errorf("snapshot size %dx%d is not valid. Try: send the terminal size", cols, rows)
	}
	var p [8]byte
	binary.LittleEndian.PutUint16(p[0:], clamp16(cols))
	binary.LittleEndian.PutUint16(p[2:], clamp16(rows))
	binary.LittleEndian.PutUint32(p[4:], uint32(max(maxLines, 0)))
	v.mu.Lock()
	v.cols, v.rows = cols, rows
	v.mu.Unlock()
	out, upTo, err := v.request(vtFrameSnapshot, p[:], timeout)
	if err != nil {
		return nil, 0, err
	}
	if len(out) == 0 {
		return nil, 0, errors.New("the shadow terminal could not make a snapshot. Try: attach again")
	}
	return out, upTo, nil
}

// AnswerQueries turns on or off the shadow terminal's replies to the
// agent's terminal queries (OnWrite). Turn it on while no client is
// attached, and off while one is, so the client's terminal answers.
func (v *VT) AnswerQueries(on bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.answering = on
	if v.child != nil {
		v.child.push(v.queriesFrame())
	}
}

// SetProfile sets what the shadow terminal reports while it answers queries.
func (v *VT) SetProfile(p VTProfile) {
	b, err := json.Marshal(p)
	if err != nil {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.profile = b
	if v.child != nil {
		v.child.push(vtFrame(vtFrameProfile, b))
	}
}

// Close stops the shadow terminal for good.
func (v *VT) Close() error {
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return nil
	}
	v.closed = true
	close(v.done)
	c := v.child
	v.child = nil
	if c != nil {
		c.fail(ErrVTDown)
	}
	v.mu.Unlock()
	if c != nil {
		// End of input makes the child exit. Kill it if it does not.
		_ = c.stdin.Close()
		select {
		case <-c.gone:
		case <-time.After(time.Second):
			c.kill()
			<-c.gone
		}
	}
	return nil
}

// dump returns a JSON description of the screen, for tests.
func (v *VT) dump(maxLines int, timeout time.Duration) ([]byte, error) {
	var p [4]byte
	binary.LittleEndian.PutUint32(p[:], uint32(max(maxLines, 0)))
	out, _, err := v.request(vtFrameDump, p[:], timeout)
	return out, err
}

// sync returns once the child has handled everything sent before it.
func (v *VT) sync(timeout time.Duration) error {
	_, _, err := v.request(vtFrameSync, nil, timeout)
	return err
}

// request sends a frame and waits for the reply of the same type. Replies
// come in order, so each one goes to the oldest waiter.
// It also returns the stream offset the child has been fed up to when it
// gets the request.
func (v *VT) request(kind byte, payload []byte, timeout time.Duration) ([]byte, int64, error) {
	ch := make(chan vtReply, 1)
	v.mu.Lock()
	c := v.child
	if c == nil || v.closed {
		v.mu.Unlock()
		return nil, 0, ErrVTDown
	}
	fed := v.fed
	c.waiters[kind] = append(c.waiters[kind], ch)
	c.push(vtFrame(kind, payload))
	v.mu.Unlock()

	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case r := <-ch:
		return r.p, fed, r.err
	case <-t.C:
		return nil, 0, fmt.Errorf("the shadow terminal did not answer within %v. Try: attach again", timeout)
	}
}

// start reads the history and starts a child that is fed it. It is used
// for the first child and for each restart.
func (v *VT) start() error {
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return ErrVTDown
	}
	v.collecting = true
	v.pending = nil
	cols, rows := v.cols, v.rows
	v.mu.Unlock()

	var hist []byte
	var start int64
	if v.opt.History != nil {
		hist, start = v.opt.History()
	}
	end := start + int64(len(hist))
	c, err := v.spawn(cols, rows)

	v.mu.Lock()
	defer v.mu.Unlock()
	pending := v.pending
	v.collecting = false
	v.pending = nil
	if err != nil {
		return err
	}
	if v.closed {
		c.kill()
		return ErrVTDown
	}
	if c.dead {
		return fmt.Errorf("the shadow terminal %s exited at once. Try: run it by hand to see why", v.opt.Path)
	}
	c.limit = v.maxBehind + len(hist)
	v.fed = max(v.fed, end)

	// The size and profile come first, so the history lays out as it did.
	c.push(v.resizeFrame())
	if v.profile != nil {
		c.push(vtFrame(vtFrameProfile, v.profile))
	}
	for p := hist; len(p) > 0; {
		n := min(len(p), vtChunk)
		c.push(vtFrame(vtFrameData, p[:n]))
		p = p[n:]
	}
	// Then what arrived while the history was read, after its end.
	for _, ch := range pending {
		d := ch.data
		if ch.off < end {
			skip := end - ch.off
			if skip >= int64(len(d)) {
				continue
			}
			d = d[skip:]
		}
		c.push(vtFrame(vtFrameData, d))
	}
	// The history raised its events once already, so they are dropped
	// until the child has caught up. Its queries were answered then too,
	// so answering only starts after it.
	if len(hist) > 0 {
		c.muted = true
		c.waiters[vtFrameSync] = append(c.waiters[vtFrameSync], nil)
		c.push(vtFrame(vtFrameSync, nil))
	}
	c.push(v.queriesFrame())
	v.child = c
	return nil
}

// spawn starts the process and its goroutines.
func (v *VT) spawn(cols, rows int) (*vtChild, error) {
	args := []string{"--cols", strconv.Itoa(cols), "--rows", strconv.Itoa(rows)}
	if v.opt.Scrollback > 0 {
		args = append(args, "--scrollback", strconv.Itoa(v.opt.Scrollback))
	}
	cmd := exec.Command(v.opt.Path, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if v.opt.Logf != nil {
		cmd.Stderr = &vtLogWriter{logf: v.opt.Logf}
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start the shadow terminal %s: %w. Try: reinstall the host helper", v.opt.Path, err)
	}
	c := &vtChild{
		cmd:     cmd,
		stdin:   stdin,
		started: time.Now(),
		wake:    make(chan struct{}, 1),
		gone:    make(chan struct{}),
		waiters: map[byte][]chan vtReply{},
	}
	go v.writeLoop(c)
	go v.readLoop(c, stdout)
	return c, nil
}

// writeLoop sends queued frames to the child's stdin.
func (v *VT) writeLoop(c *vtChild) {
	w := bufio.NewWriterSize(c.stdin, 64<<10)
	for {
		v.mu.Lock()
		for len(c.queue) == 0 && !c.dead {
			v.mu.Unlock()
			select {
			case <-c.wake:
			case <-c.gone:
				return
			}
			v.mu.Lock()
		}
		if c.dead {
			v.mu.Unlock()
			return
		}
		batch := c.queue
		c.queue = nil
		v.mu.Unlock()

		n := 0
		for _, f := range batch {
			if _, err := w.Write(f); err != nil {
				c.kill()
				return
			}
			n += len(f)
		}
		if err := w.Flush(); err != nil {
			c.kill()
			return
		}
		v.mu.Lock()
		c.queued -= n
		v.mu.Unlock()
	}
}

// readLoop handles the child's frames until it exits, then restarts it.
func (v *VT) readLoop(c *vtChild, stdout io.Reader) {
	r := bufio.NewReaderSize(stdout, 64<<10)
	var err error
	for {
		var hdr [5]byte
		if _, err = io.ReadFull(r, hdr[:]); err != nil {
			break
		}
		n := binary.LittleEndian.Uint32(hdr[1:])
		p := make([]byte, n)
		if _, err = io.ReadFull(r, p); err != nil {
			break
		}
		v.handle(c, hdr[0], p)
	}
	werr := c.cmd.Wait()
	if werr != nil {
		err = werr
	}
	v.exited(c, err)
}

func (v *VT) handle(c *vtChild, kind byte, p []byte) {
	switch kind {
	case vtFrameWrite:
		if f := v.opt.OnWrite; f != nil {
			f(p)
		}
	case vtFrameEvent:
		v.mu.Lock()
		muted := c.muted
		v.mu.Unlock()
		if f := v.opt.OnEvent; f != nil && !muted {
			var e VTEvent
			if json.Unmarshal(p, &e) == nil {
				f(e)
			}
		}
	case vtFrameSnapshot, vtFrameDump, vtFrameSync:
		v.mu.Lock()
		q := c.waiters[kind]
		var ch chan vtReply
		if len(q) > 0 {
			ch = q[0]
			c.waiters[kind] = q[1:]
		}
		if ch == nil && kind == vtFrameSync {
			// The history replay has finished.
			c.muted = false
		}
		v.mu.Unlock()
		if ch != nil {
			ch <- vtReply{p: p}
		}
	}
}

// exited runs once the child's process has exited.
func (v *VT) exited(c *vtChild, err error) {
	v.mu.Lock()
	c.dead = true
	close(c.gone)
	c.fail(ErrVTDown)
	if v.child != c || v.closed {
		v.mu.Unlock()
		return
	}
	v.child = nil
	if time.Since(c.started) > vtStableAfter {
		v.crashes = 0
	}
	wait := vtBackoff[min(v.crashes, len(vtBackoff)-1)]
	v.crashes++
	v.mu.Unlock()

	if err == nil {
		err = errors.New("it closed its output")
	}
	v.logf("shadow terminal stopped (%v): restarting it in %v", err, wait)
	go v.restart(wait)
}

func (v *VT) restart(wait time.Duration) {
	for {
		select {
		case <-v.done:
			return
		case <-time.After(wait):
		}
		err := v.start()
		if err == nil || errors.Is(err, ErrVTDown) {
			return
		}
		v.mu.Lock()
		wait = vtBackoff[min(v.crashes, len(vtBackoff)-1)]
		v.crashes++
		v.mu.Unlock()
		v.logf("shadow terminal did not start (%v): trying again in %v", err, wait)
	}
}

func (v *VT) resizeFrame() []byte {
	var p [8]byte
	binary.LittleEndian.PutUint16(p[0:], clamp16(v.cols))
	binary.LittleEndian.PutUint16(p[2:], clamp16(v.rows))
	binary.LittleEndian.PutUint16(p[4:], clamp16(v.xpx))
	binary.LittleEndian.PutUint16(p[6:], clamp16(v.ypx))
	return vtFrame(vtFrameResize, p[:])
}

func (v *VT) queriesFrame() []byte {
	on := byte(0)
	if v.answering {
		on = 1
	}
	return vtFrame(vtFrameQueries, []byte{on})
}

func (v *VT) logf(format string, args ...any) {
	if v.opt.Logf != nil {
		v.opt.Logf(format, args...)
	}
}

// push queues a frame for the writer. VT.mu is held.
func (c *vtChild) push(f []byte) {
	if c.dead {
		return
	}
	c.queue = append(c.queue, f)
	c.queued += len(f)
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// fail ends every request still waiting for a reply. VT.mu is held.
func (c *vtChild) fail(err error) {
	for k, q := range c.waiters {
		for _, ch := range q {
			if ch != nil {
				ch <- vtReply{err: err}
			}
		}
		delete(c.waiters, k)
	}
}

func (c *vtChild) kill() {
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

// vtFrame encodes one frame.
func vtFrame(kind byte, payload []byte) []byte {
	f := make([]byte, 5+len(payload))
	f[0] = kind
	binary.LittleEndian.PutUint32(f[1:], uint32(len(payload)))
	copy(f[5:], payload)
	return f
}

func clamp16(n int) uint16 {
	return uint16(min(max(n, 0), 0xffff))
}

// vtLogWriter passes the child's stderr to Logf, a line at a time.
type vtLogWriter struct {
	logf func(format string, args ...any)
	buf  []byte
}

func (w *vtLogWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		w.logf("shadow terminal: %s", w.buf[:i])
		w.buf = w.buf[i+1:]
	}
	if len(w.buf) > 4096 {
		w.logf("shadow terminal: %s", w.buf)
		w.buf = nil
	}
	return len(p), nil
}
