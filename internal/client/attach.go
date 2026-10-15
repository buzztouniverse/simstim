package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/muesli/cancelreader"
	"golang.org/x/sys/unix"
	xterm "golang.org/x/term"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/keys"
	"github.com/iAmLakshya/simstim/internal/proto"
	"github.com/iAmLakshya/simstim/internal/term"
)

// AttachState is what the client remembers about the last session attached in
// this terminal, so the next attach can continue from the exact byte the
// terminal already shows instead of redrawing everything.
type AttachState struct {
	Key    string // host/session
	Offset int64
	Modes  term.Modes
	Valid  bool
}

// AttachOptions configures one attach.
type AttachOptions struct {
	Host     Host
	Session  string
	Title    string
	Settings Settings
	State    *AttachState
	// Interactive allows SSH prompts on the first connection.
	Interactive bool
	In, Out     *os.File
}

// AttachResult says how an attach ended.
type AttachResult struct {
	Detached   bool
	Ended      *proto.Ended
	Missed     int64 // bytes replayed after reconnects
	Reconnects int
}

// Heartbeat timings.
const (
	pingEvery   = 5 * time.Second
	deadAfter   = 15 * time.Second
	bannerAfter = 3 * time.Second
	doubleTap   = 400 * time.Millisecond
	escFlush    = 15 * time.Millisecond
)

type attacher struct {
	o       AttachOptions
	conn    *Conn
	ssh     *SSH
	fd      int
	tracker *term.Tracker
	san     term.Sanitizer
	live    int64
	offset  int64
	filter  *keys.Filter
	inCh    chan []byte
	res     AttachResult
	key     string
	// swallow holds watched keys whose press we consumed, so their release is swallowed too.
	swallow map[int]bool
}

// Attach connects to a session and runs it in this terminal until the user
// detaches, the session ends, or ctx is cancelled. The session keeps running.
func Attach(ctx context.Context, o AttachOptions) (AttachResult, error) {
	if o.In == nil {
		o.In = os.Stdin
	}
	if o.Out == nil {
		o.Out = os.Stdout
	}
	if o.State == nil {
		o.State = &AttachState{}
	}
	dk, err := keys.Parse(o.Settings.Attach.DetachKey)
	if err != nil {
		return AttachResult{}, &UserError{Msg: "bad detach key: " + err.Error(), Hint: "Fix attach.detach_key in the config."}
	}
	pk, err := keys.Parse(o.Settings.Attach.ImagePasteKey)
	if err != nil {
		return AttachResult{}, &UserError{Msg: "bad image paste key: " + err.Error(), Hint: "Fix attach.image_paste_key in the config."}
	}
	a := &attacher{o: o, ssh: &SSH{Host: o.Host}, fd: int(o.In.Fd()), filter: keys.New(dk, pk),
		key: o.Host.Name + "/" + o.Session, swallow: map[int]bool{}}

	last := int64(-1)
	if o.State.Valid && o.State.Key == a.key {
		last = o.State.Offset
	}
	conn, err := Dial(ctx, o.Host, o.Interactive)
	if err != nil {
		return AttachResult{}, err
	}
	a.conn = conn
	defer func() { a.conn.Close() }()
	cols, rows, xpx, ypx := a.size()
	att, err := a.handshake(last, cols, rows, xpx, ypx)
	if err != nil {
		return AttachResult{}, err
	}

	old, err := xterm.MakeRaw(a.fd)
	if err != nil {
		return AttachResult{}, fmt.Errorf("cannot put the terminal in raw mode: %w", err)
	}
	cr, err := cancelreader.NewReader(o.In)
	if err != nil {
		_ = xterm.Restore(a.fd, old)
		return AttachResult{}, err
	}
	a.inCh = make(chan []byte, 256)
	go func() {
		buf := make([]byte, 32<<10)
		for {
			n, err := cr.Read(buf)
			if n > 0 {
				a.inCh <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				close(a.inCh)
				return
			}
		}
	}()

	// Remember the title, and show which session this terminal now shows.
	a.write([]byte("\x1b[22;0t"))
	if o.Title != "" {
		a.write([]byte("\x1b]2;" + o.Title + "\x1b\\"))
	}
	a.begin(att, o.State, rows)

	runErr := a.loop(ctx)

	// Put the terminal back exactly as we found it: undo the session's modes,
	// never with RIS, which would wipe the scrollback.
	cr.Cancel()
	a.write(a.tracker.M.ResetSequence())
	a.write([]byte("\x1b]9;4;0\x1b\\\x1b[23;0t"))
	_ = xterm.Restore(a.fd, old)
	_ = cr.Close()

	o.State.Key, o.State.Offset, o.State.Modes, o.State.Valid = a.key, a.offset, a.tracker.M, true
	if a.res.Ended != nil {
		o.State.Valid = false
	}
	return a.res, runErr
}

func (a *attacher) write(b []byte) {
	if len(b) > 0 {
		_, _ = a.o.Out.Write(b)
	}
}

func (a *attacher) size() (cols, rows, xpx, ypx int) {
	ws, err := unix.IoctlGetWinsize(int(a.o.Out.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 {
		return 80, 24, 0, 0
	}
	return int(ws.Col), int(ws.Row), int(ws.Xpixel), int(ws.Ypixel)
}

// handshake sends the attach request and waits for the answer.
func (a *attacher) handshake(last int64, cols, rows, xpx, ypx int) (proto.Attached, error) {
	req := proto.Attach{Session: a.o.Session, Cols: cols, Rows: rows, XPx: xpx, YPx: ypx, Client: brand.Name, LastOffset: last}
	if err := a.conn.SendJSON(proto.TAttach, req); err != nil {
		return proto.Attached{}, ErrClosed
	}
	timeout := time.After(15 * time.Second)
	for {
		select {
		case f := <-a.conn.Stream():
			switch f.Type {
			case proto.TAttached:
				var att proto.Attached
				if err := json.Unmarshal(f.Payload, &att); err != nil {
					return att, err
				}
				return att, nil
			case proto.TError:
				var pe proto.Error
				_ = json.Unmarshal(f.Payload, &pe)
				return proto.Attached{}, &UserError{Msg: pe.Message, Hint: pe.Hint}
			}
		case <-a.conn.Done():
			return proto.Attached{}, ErrClosed
		case <-timeout:
			return proto.Attached{}, errors.New("the host did not answer the attach in time")
		}
	}
}

// begin prepares the screen for the bytes that follow.
func (a *attacher) begin(att proto.Attached, st *AttachState, rows int) {
	a.live = att.Live
	a.san = term.Sanitizer{}
	if att.Mode == "replay" && st.Valid && st.Key == a.key {
		// The terminal still shows this session up to st.Offset: put its modes back and continue.
		a.tracker = &term.Tracker{M: st.Modes}
		a.offset = att.From
		a.write(st.Modes.RestoreSequence())
		return
	}
	// A full redraw. Keep what is on screen in the scrollback, then start clean.
	a.tracker = term.NewTracker()
	a.offset = att.From
	if a.o.Settings.Attach.Scrollback == "clear" {
		a.write([]byte("\x1b[H\x1b[2J\x1b[3J"))
	} else {
		a.write([]byte(fmt.Sprintf("\x1b[%d;1H%s\x1b[H", rows, strings.Repeat("\n", rows))))
	}
}

// output shows bytes from offset off, skipping any the terminal already has.
func (a *attacher) output(off int64, data []byte) {
	end := off + int64(len(data))
	if end <= a.offset {
		return
	}
	if off < a.offset {
		data = data[a.offset-off:]
		off = a.offset
	}
	w := data
	if off < a.live {
		n := a.live - off
		if n > int64(len(data)) {
			n = int64(len(data))
		}
		hist := a.san.Feed(data[:n])
		if off+n >= a.live {
			hist = append(hist, a.san.Flush()...)
		}
		w = append(hist, data[n:]...)
	}
	a.write(w)
	a.tracker.Feed(data)
	a.offset = end
}

func (a *attacher) loop(ctx context.Context) error {
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	heart := time.NewTicker(pingEvery)
	defer heart.Stop()
	var detachTimer, flushTimer <-chan time.Time
	for {
		select {
		case f := <-a.conn.Stream():
			if done, err := a.frame(f); done || err != nil {
				return err
			}
		case b, ok := <-a.inCh:
			if !ok {
				a.res.Detached = true
				return nil
			}
			if a.input(b, &detachTimer) {
				return nil
			}
			if a.filter.Pending() {
				flushTimer = time.After(escFlush)
			}
		case <-flushTimer:
			flushTimer = nil
			if b := a.filter.Flush(); len(b) > 0 {
				_ = a.conn.Send(proto.TInput, b)
			}
		case <-detachTimer:
			a.res.Detached = true
			return nil
		case <-winch:
			c, r, x, y := a.size()
			_ = a.conn.SendJSON(proto.TResize, proto.Resize{Cols: c, Rows: r, XPx: x, YPx: y})
		case <-heart.C:
			_ = a.conn.Ping()
			if time.Since(a.conn.LastRecv()) > deadAfter {
				a.conn.kill()
			}
		case <-a.conn.Done():
			if detached, err := a.reconnect(ctx); detached || err != nil {
				a.res.Detached = detached
				return err
			}
		case <-ctx.Done():
			a.res.Detached = true
			return nil
		}
	}
}

// frame handles one frame from the host. done is true when the attach is over.
func (a *attacher) frame(f proto.Frame) (done bool, err error) {
	switch f.Type {
	case proto.TOutput:
		off, data, err := proto.DecodeOutput(f.Payload)
		if err == nil {
			a.output(off, data)
		}
	case proto.TEnded:
		var e proto.Ended
		_ = json.Unmarshal(f.Payload, &e)
		a.res.Ended = &e
		return true, nil
	case proto.TError:
		var pe proto.Error
		_ = json.Unmarshal(f.Payload, &pe)
		if pe.Code == "session_gone" {
			a.res.Ended = &proto.Ended{ExitCode: -1}
			return true, nil
		}
		return true, &UserError{Msg: pe.Message, Hint: pe.Hint}
	}
	return false, nil
}

// input handles keys typed by the user. It returns true to detach now.
func (a *attacher) input(b []byte, detachTimer *<-chan time.Time) bool {
	for _, p := range a.filter.Feed(b) {
		switch p.Kind {
		case keys.Pass:
			_ = a.conn.Send(proto.TInput, p.Raw)
		case keys.Paste:
			a.paste(p)
		case keys.Match:
			switch p.Index {
			case 0: // detach
				if p.Release || p.Repeat {
					if !a.swallow[0] {
						_ = a.conn.Send(proto.TInput, p.Raw)
					}
					continue
				}
				if *detachTimer != nil {
					// A second press: send the key itself to the agent.
					*detachTimer = nil
					a.swallow[0] = false
					_ = a.conn.Send(proto.TInput, p.Raw)
					continue
				}
				a.swallow[0] = true
				*detachTimer = time.After(doubleTap)
			case 1: // image paste
				if p.Release || p.Repeat {
					if !a.swallow[1] {
						_ = a.conn.Send(proto.TInput, p.Raw)
					}
					continue
				}
				img := ClipboardImage()
				if img == nil {
					a.swallow[1] = false
					_ = a.conn.Send(proto.TInput, p.Raw)
					continue
				}
				a.swallow[1] = true
				a.sendUpload("clipboard.png", img)
			}
		}
	}
	return false
}

// sendUpload uploads a file and pastes its remote path into the session.
func (a *attacher) sendUpload(name string, data []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	path, err := a.conn.Upload(ctx, a.o.Session, name, data)
	if err != nil {
		a.flash("upload failed: " + err.Error())
		return
	}
	a.pasteText([]byte(path + " "))
}

// pasteText sends text the way a paste would arrive.
func (a *attacher) pasteText(t []byte) {
	if a.tracker.M.Private[2004] {
		t = append(append([]byte("\x1b[200~"), t...), []byte("\x1b[201~")...)
	}
	_ = a.conn.Send(proto.TInput, t)
}

// paste forwards a bracketed paste. When it is only local file paths (a file
// dropped on the terminal), the files are uploaded and the paths rewritten.
func (a *attacher) paste(p keys.Piece) {
	files := localFiles(p.Text)
	if len(files) == 0 {
		_ = a.conn.Send(proto.TInput, p.Raw)
		return
	}
	var remote []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil || len(data) > MaxImage {
			_ = a.conn.Send(proto.TInput, p.Raw)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		path, err := a.conn.Upload(ctx, a.o.Session, filepath.Base(f), data)
		cancel()
		if err != nil {
			a.flash("upload failed: " + err.Error())
			_ = a.conn.Send(proto.TInput, p.Raw)
			return
		}
		remote = append(remote, shellQuote(path))
	}
	a.pasteText([]byte(strings.Join(remote, " ") + " "))
}

// localFiles returns the paths in t if every token is an existing local regular file.
func localFiles(t []byte) []string {
	toks := splitShellWords(strings.TrimSpace(string(t)))
	if len(toks) == 0 || len(toks) > 20 {
		return nil
	}
	for _, s := range toks {
		if !filepath.IsAbs(s) {
			return nil
		}
		st, err := os.Stat(s)
		if err != nil || !st.Mode().IsRegular() {
			return nil
		}
	}
	return toks
}

// splitShellWords splits on unescaped spaces, handling backslash escapes and quotes,
// the way terminals quote dropped paths.
func splitShellWords(s string) []string {
	var out []string
	var cur bytes.Buffer
	var quote rune
	esc, have := false, false
	for _, r := range s {
		switch {
		case esc:
			cur.WriteRune(r)
			esc, have = false, true
		case r == '\\' && quote != '\'':
			esc = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, have = r, true
		case r == ' ' || r == '\n' || r == '\t':
			if have {
				out = append(out, cur.String())
				cur.Reset()
				have = false
			}
		default:
			cur.WriteRune(r)
			have = true
		}
	}
	if have {
		out = append(out, cur.String())
	}
	return out
}

func shellQuote(p string) string {
	if strings.ContainsAny(p, " '\"\\$`") {
		return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
	}
	return p
}

// flash shows a short message on the bottom row without disturbing the session.
func (a *attacher) flash(msg string) {
	_, rows, _, _ := a.size()
	a.write([]byte(fmt.Sprintf("\x1b7\x1b[%d;1H\x1b[2K\x1b[2m %s · %s\x1b[0m\x1b8", rows, brand.DisplayName, msg)))
}

func (a *attacher) clearFlash() {
	_, rows, _, _ := a.size()
	a.write([]byte(fmt.Sprintf("\x1b7\x1b[%d;1H\x1b[2K\x1b8", rows)))
}

// reconnect keeps trying to reach the session again. It returns detached=true
// if the user pressed the detach key meanwhile.
func (a *attacher) reconnect(ctx context.Context) (detached bool, err error) {
	a.res.Reconnects++
	a.write([]byte("\x1b]9;4;3\x1b\\\x1b]2;⧗ reconnecting · " + a.key + "\x1b\\"))
	start := time.Now()
	banner := time.After(bannerAfter)
	delay := time.Duration(0)
	for {
		retry := time.After(delay)
	wait:
		for {
			select {
			case <-banner:
				a.flash("connection lost · reconnecting · keys are not sent · " + a.o.Settings.Attach.DetachKey + " to leave")
			case b, ok := <-a.inCh:
				if !ok {
					return true, nil
				}
				for _, p := range a.filter.Feed(b) {
					if p.Kind == keys.Match && p.Index == 0 && !p.Release && !p.Repeat {
						return true, nil
					}
				}
				_ = a.filter.Flush()
				break wait // a key press: try again now
			case <-retry:
				break wait
			case <-ctx.Done():
				return true, nil
			}
		}
		a.ssh.ExitMaster()
		dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		conn, err := Dial(dctx, a.o.Host, false)
		cancel()
		if err == nil {
			a.conn.Close()
			a.conn = conn
			cols, rows, xpx, ypx := a.size()
			before := a.offset
			att, herr := a.handshake(a.offset, cols, rows, xpx, ypx)
			if herr == nil {
				a.clearFlash()
				if att.Mode == "replay" {
					a.live = att.Live
					a.san = term.Sanitizer{}
				} else {
					a.begin(att, &AttachState{}, rows)
				}
				a.res.Missed += att.Live - before
				title := []byte("\x1b]2;" + a.o.Title + "\x1b\\")
				if len(a.tracker.M.Title) > 0 {
					title = a.tracker.M.Title
				}
				a.write(append([]byte("\x1b]9;4;0\x1b\\"), title...))
				return false, nil
			}
			var ue *UserError
			if errors.As(herr, &ue) {
				return false, herr
			}
		} else {
			var ue *UserError
			if errors.As(err, &ue) {
				return false, err
			}
		}
		if time.Since(start) > 0 {
			switch {
			case delay == 0:
				delay = time.Second
			case delay < 30*time.Second:
				delay *= 2
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
			}
		}
	}
}
