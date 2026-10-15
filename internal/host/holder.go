package host

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"

	"github.com/iAmLakshya/simstim/internal/agents"
	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/paths"
	"github.com/iAmLakshya/simstim/internal/proto"
	"github.com/iAmLakshya/simstim/internal/ring"
	"github.com/iAmLakshya/simstim/internal/version"
)

// RingSize is how much raw output each session keeps for replay.
const RingSize = 8 << 20

// clientQueue is how many output frames may wait for a slow client
// before it is dropped. A dropped client reconnects and catches up by offset.
const clientQueue = 4096

type outFrame struct {
	t proto.Type
	p []byte
}

type hclient struct {
	conn     net.Conn
	w        *proto.Writer
	out      chan outFrame
	attached bool
	once     sync.Once
	done     chan struct{}
}

func (c *hclient) close() {
	c.once.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}

// Holder owns one session: its PTY, output ring, socket and manifest.
type Holder struct {
	id   string
	log  *log.Logger
	ln   net.Listener
	ptmx *os.File
	fd   int
	cmd  *exec.Cmd

	mu        sync.Mutex
	m         *Manifest
	ring      *ring.Ring
	clients   map[*hclient]struct{}
	leader    *hclient
	scan      scanner
	lastOut   time.Time
	needInput bool
	busy      bool
	exited    bool
	ended     proto.Ended
	dirty     bool
	oomBase   int64
	// stopping is true while the system stops the session (for example at shutdown).
	stopping bool
	// From agent hooks.
	hookSeen    bool
	hookBusy    bool
	lastTurnEnd time.Time
	notifyMsg   string
	// The shadow terminal, for snapshots and query answers; nil if unavailable.
	vt     *VT
	ptyOut chan []byte // ordered writes to the PTY from the shadow terminal
	// From screen rules.
	scr       *screenModel
	ruleState string
	ruleID    string
}

// RunHolder runs the holder for session id until the session is killed.
func RunHolder(id string) error {
	m, err := LoadManifest(id)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}
	lf, err := os.OpenFile(paths.HolderLog(id), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer lf.Close()
	h := &Holder{
		id:      id,
		m:       m,
		log:     log.New(lf, "", log.LstdFlags|log.Lmicroseconds),
		ring:    ring.New(RingSize),
		clients: map[*hclient]struct{}{},
	}
	h.log.Printf("holder %s starting (%s %s)", id, brand.HostBinary, version.Version)
	if err := h.listen(); err != nil {
		h.log.Printf("listen: %v", err)
		return err
	}
	if err := h.start(); err != nil {
		h.log.Printf("start agent: %v", err)
		h.mu.Lock()
		code := 127
		h.m.Status = Status{State: StateErrored, Since: time.Now().UTC(), ExitCode: &code, Reason: err.Error()}
		h.mu.Unlock()
		_ = h.m.Save()
		h.ln.Close()
		os.Remove(paths.Socket(id))
		return err
	}
	h.startVT()
	go h.acceptLoop()
	go h.readLoop()
	go h.waitLoop()
	go h.tick()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	s := <-sig
	h.log.Printf("signal %v: stopping", s)
	// A signal means the system (or the user's manager) is stopping the
	// session, for example at shutdown. The agent did not end by itself, so the
	// session stays eligible for restore after the restart.
	h.stop("stopped with the host", true)
	return nil
}

func (h *Holder) listen() error {
	sock := paths.Socket(h.id)
	if err := paths.EnsurePrivateDir(filepath.Dir(sock)); err != nil {
		return err
	}
	if err := paths.EnsurePrivateDir(filepath.Dir(filepath.Dir(sock))); err != nil {
		return err
	}
	_ = os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return err
	}
	if err := os.Chmod(sock, 0o600); err != nil {
		ln.Close()
		return err
	}
	h.ln = ln
	return nil
}

func (h *Holder) start() error {
	m := h.m
	if len(m.Agent.Argv) == 0 {
		return errors.New("no command to run")
	}
	argv := m.Agent.Argv
	if m.RunHook != "" {
		// Run the setup (or resume) hook in the session's terminal, so its output
		// is visible, then the agent. A failing hook is reported, not fatal.
		script := `"$0"; s=$?; [ $s -eq 0 ] || printf '\n[hook %s exited with %s]\n\n' "$0" $s; exec "$@"`
		argv = append([]string{"/bin/sh", "-c", script, m.RunHook}, argv...)
		m.RunHook = ""
	}
	bin, err := lookPath(argv[0], envValue(m.Env, "PATH"))
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, argv[1:]...)
	cmd.Args[0] = argv[0]
	cmd.Env = m.Env
	cmd.Dir = m.Cwd
	cols, rows := m.Term.Cols, m.Term.Rows
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}
	h.ptmx = ptmx
	h.fd = int(ptmx.Fd())
	if _, ok := agents.Find(m.Agent.Kind); ok {
		h.scr = newScreenModel(cols, rows)
	}
	h.cmd = cmd
	// Make the agent, not the holder, the first choice of the OOM killer.
	_ = os.WriteFile(fmt.Sprintf("/proc/%d/oom_score_adj", cmd.Process.Pid), []byte("500"), 0)
	h.oomBase = readOOMKills()

	h.mu.Lock()
	now := time.Now().UTC()
	if u := selfUnit(); u != "" {
		m.Run.Unit = u
	}
	m.Run.HolderPID = os.Getpid()
	m.Run.AgentPID = cmd.Process.Pid
	m.Run.Socket = paths.Socket(h.id)
	m.Run.BootID = BootID()
	m.Run.StartedAt = now
	if exe, err := os.Executable(); err == nil {
		m.HolderBinary = exe
	}
	m.HostVersion = version.Version
	m.Status = Status{State: StateWorking, Since: now, Source: "start"}
	h.lastOut = time.Now()
	h.mu.Unlock()
	if err := m.Save(); err != nil {
		h.log.Printf("save manifest: %v", err)
	}
	h.log.Printf("agent %v started, pid %d", m.Agent.Argv, cmd.Process.Pid)
	return nil
}

func (h *Holder) readLoop() {
	buf := make([]byte, 64<<10)
	for {
		n, err := h.ptmx.Read(buf)
		if n > 0 {
			h.output(buf[:n])
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, syscall.EIO) {
				h.log.Printf("pty read: %v", err)
			}
			return
		}
	}
}

func (h *Holder) output(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	off := h.ring.End()
	h.ring.Write(data)
	if h.vt != nil {
		h.vt.Feed(off, data)
	}
	payload := proto.EncodeOutput(off, data)
	for c := range h.clients {
		if !c.attached {
			continue
		}
		select {
		case c.out <- outFrame{proto.TOutput, payload}:
		default:
			h.log.Printf("client too slow, dropping it")
			go c.close()
		}
	}
	h.lastOut = time.Now()
	h.m.LastActivityAt = time.Now().UTC()
	h.scr.feed(data)
	h.scan.feed(data, func(s outSignal) {
		switch s {
		case sigTitle:
			if h.scr != nil {
				h.scr.title = h.scan.title
				h.scr.dirty = true
			}
		case sigBell, sigNotify:
			h.needInput = true
		case sigProgressBusy:
			h.busy = true
		case sigProgressDone:
			h.busy = false
		}
	})
	h.dirty = true
}

func (h *Holder) waitLoop() {
	err := h.cmd.Wait()
	ended := proto.Ended{}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				ended.Signal = ws.Signal().String()
				ended.ExitCode = 128 + int(ws.Signal())
				if ws.Signal() == syscall.SIGKILL && readOOMKills() > h.oomBase {
					ended.OOM = true
				}
			} else {
				ended.ExitCode = ee.ExitCode()
			}
		} else {
			ended.ExitCode = 1
		}
	}
	h.log.Printf("agent exited: code %d signal %q oom %v", ended.ExitCode, ended.Signal, ended.OOM)
	h.mu.Lock()
	h.exited = true
	h.ended = ended
	if h.stopping {
		// The holder stopped the agent because the system is stopping: keep
		// the session's state, so it is restored after the restart.
		h.mu.Unlock()
		return
	}
	code := ended.ExitCode
	st := Status{State: StateExited, Since: time.Now().UTC(), ExitCode: &code, Signal: ended.Signal, OOM: ended.OOM, Source: "process"}
	if ended.OOM {
		st.State = StateErrored
		st.Reason = "oom"
	} else if code != 0 {
		st.State = StateErrored
	}
	h.m.Status = st
	b, _ := json.Marshal(ended)
	for c := range h.clients {
		if c.attached {
			select {
			case c.out <- outFrame{proto.TEnded, b}:
			default:
			}
		}
	}
	h.mu.Unlock()
	_ = h.m.Save()
}

// tick updates the status and saves the manifest when it changed.
func (h *Holder) tick() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		h.mu.Lock()
		if !h.exited {
			if h.scr != nil && h.scr.dirty {
				h.scr.dirty = false
				h.ruleState, h.ruleID, _ = agents.Detect(h.m.Agent.Kind, h.scr.screen())
			}
			state := StateIdle
			src := "quiet"
			switch {
			case h.ruleState == StateNeedsInput:
				state, src = StateNeedsInput, "screen:"+h.ruleID
			case h.needInput:
				state, src = StateNeedsInput, "notify"
			case h.ruleState == StateWorking:
				state, src = StateWorking, "screen:"+h.ruleID
			case h.hookBusy:
				state, src = StateWorking, "hook"
			case h.busy:
				state, src = StateWorking, "progress"
			case h.hookSeen && !h.lastTurnEnd.IsZero() && time.Since(h.lastTurnEnd) < 3*time.Second:
				state, src = StateIdle, "hook"
			case time.Since(h.lastOut) < 3*time.Second:
				state, src = StateWorking, "output"
			}
			if state != h.m.Status.State {
				oom, reason := h.m.Status.OOM, h.m.Status.Reason
				h.m.Status = Status{State: state, Since: time.Now().UTC(), Source: src}
				if oom {
					h.m.Status.OOM, h.m.Status.Reason = true, reason
				} else if state == StateNeedsInput && h.notifyMsg != "" {
					h.m.Status.Reason = h.notifyMsg
				}
				h.dirty = true
			}
		}
		if n := readOOMKills(); n > h.oomBase && !h.m.Status.OOM {
			// A process in the session hit the memory cap and was stopped.
			h.m.Status.OOM = true
			h.m.Status.Reason = "a process in this session ran out of memory and was stopped"
			h.needInput = true
			h.dirty = true
		}
		dirty := h.dirty
		h.dirty = false
		var snapshot Manifest
		if dirty {
			snapshot = *h.m
		}
		h.mu.Unlock()
		if dirty {
			if err := snapshot.Save(); err != nil {
				h.log.Printf("save manifest: %v", err)
			}
		}
	}
}

func (h *Holder) acceptLoop() {
	for {
		conn, err := h.ln.Accept()
		if err != nil {
			return
		}
		go h.serve(conn)
	}
}

func (h *Holder) serve(conn net.Conn) {
	c := &hclient{conn: conn, w: proto.NewWriter(conn), out: make(chan outFrame, clientQueue), done: make(chan struct{})}
	defer func() {
		h.drop(c)
		c.close()
	}()
	r := bufio.NewReaderSize(conn, 64<<10)
	f, err := proto.Read(r)
	if err != nil || f.Type != proto.THello {
		return
	}
	var hello proto.Hello
	if json.Unmarshal(f.Payload, &hello) != nil || hello.Check(version.Protocol) != nil {
		_ = c.w.WriteJSON(proto.TError, proto.Error{Code: "protocol", Message: "incompatible client"})
		return
	}
	if err := c.w.WriteJSON(proto.THello, proto.Hello{Magic: proto.Magic, Protocol: version.Protocol, Version: version.Version, Role: "holder"}); err != nil {
		return
	}
	for {
		f, err := proto.Read(r)
		if err != nil {
			return
		}
		switch f.Type {
		case proto.TAttach:
			var a proto.Attach
			if json.Unmarshal(f.Payload, &a) == nil {
				h.attach(c, a)
			}
		case proto.TInput:
			h.input(c, f.Payload)
		case proto.TResize:
			var rz proto.Resize
			if json.Unmarshal(f.Payload, &rz) == nil {
				h.resize(c, rz, false)
			}
		case proto.TDetach:
			h.drop(c)
		case proto.TPing:
			_ = c.w.Write(proto.TPong, f.Payload)
		case proto.TRequest:
			var req proto.Request
			if json.Unmarshal(f.Payload, &req) == nil {
				h.request(c, req)
			}
		}
	}
}

func (h *Holder) attach(c *hclient, a proto.Attach) {
	// A client that has none of this session's output gets a snapshot from the
	// shadow terminal, taken before the lock so output keeps flowing.
	var snap []byte
	snapAt := int64(-1)
	h.mu.Lock()
	canReplay := a.LastOffset >= 0 && h.ring.Has(a.LastOffset)
	vt := h.vt
	h.mu.Unlock()
	if !canReplay && vt != nil {
		if s, upTo, err := vt.Snapshot(a.Cols, a.Rows, snapshotLines, 3*time.Second); err == nil {
			snap, snapAt = s, upTo
		} else {
			h.log.Printf("snapshot failed, replaying the ring instead: %v", err)
		}
	}
	h.mu.Lock()
	mode, from := "full", h.ring.Start()
	switch {
	case a.LastOffset >= 0 && h.ring.Has(a.LastOffset):
		mode, from = "replay", a.LastOffset
	case snapAt >= 0 && h.ring.Has(snapAt):
		mode, from = "snapshot", snapAt
	}
	data, _ := h.ring.ReadFrom(from)
	live := h.ring.End()
	c.attached = true
	h.clients[c] = struct{}{}
	h.leader = c
	exited := h.exited
	ended := h.ended
	h.m.LastAttachedAt = time.Now().UTC()
	h.dirty = true
	h.mu.Unlock()

	if h.vt != nil {
		h.vt.AnswerQueries(false) // a client's terminal answers from now on
	}
	if err := c.w.WriteJSON(proto.TAttached, proto.Attached{Session: h.id, Mode: mode, From: from, Live: live, Exited: exited}); err != nil {
		c.close()
		return
	}
	if mode == "snapshot" {
		for len(snap) > 0 {
			n := min(len(snap), 256<<10)
			if err := c.w.Write(proto.TSnapshot, snap[:n]); err != nil {
				c.close()
				return
			}
			snap = snap[n:]
		}
	}
	for off := from; len(data) > 0; {
		n := len(data)
		if n > 256<<10 {
			n = 256 << 10
		}
		if err := c.w.Write(proto.TOutput, proto.EncodeOutput(off, data[:n])); err != nil {
			c.close()
			return
		}
		off += int64(n)
		data = data[n:]
	}
	go func() {
		for {
			select {
			case f := <-c.out:
				if err := c.w.Write(f.t, f.p); err != nil {
					c.close()
					return
				}
			case <-c.done:
				return
			}
		}
	}()
	if exited {
		b, _ := json.Marshal(ended)
		c.out <- outFrame{proto.TEnded, b}
		return
	}
	h.resize(c, proto.Resize{Cols: a.Cols, Rows: a.Rows, XPx: a.XPx, YPx: a.YPx}, true)
}

func (h *Holder) input(c *hclient, p []byte) {
	h.mu.Lock()
	if h.exited || !c.attached {
		h.mu.Unlock()
		return
	}
	h.leader = c
	h.needInput = false
	h.notifyMsg = ""
	h.mu.Unlock()
	if _, err := h.ptmx.Write(p); err != nil {
		h.log.Printf("pty write: %v", err)
	}
}

func (h *Holder) resize(c *hclient, rz proto.Resize, force bool) {
	h.mu.Lock()
	leader := h.leader == c
	exited := h.exited
	if leader && rz.Cols > 0 && rz.Rows > 0 {
		h.m.Term.Cols, h.m.Term.Rows = rz.Cols, rz.Rows
		h.dirty = true
	}
	h.mu.Unlock()
	if !leader || exited || rz.Cols <= 0 || rz.Rows <= 0 {
		return
	}
	_ = pty.Setsize(h.ptmx, &pty.Winsize{Cols: uint16(rz.Cols), Rows: uint16(rz.Rows), X: uint16(rz.XPx), Y: uint16(rz.YPx)})
	h.mu.Lock()
	h.scr.resize(rz.Cols, rz.Rows)
	vt := h.vt
	h.mu.Unlock()
	if vt != nil {
		vt.Resize(rz.Cols, rz.Rows, rz.XPx, rz.YPx)
	}
	if force {
		// Make the agent repaint even when the size did not change.
		if pgrp, err := unix.IoctlGetInt(h.fd, unix.TIOCGPGRP); err == nil && pgrp > 0 {
			_ = unix.Kill(-pgrp, unix.SIGWINCH)
		}
	}
}

func (h *Holder) drop(c *hclient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	c.attached = false
	if h.leader == c {
		h.leader = nil
	}
	if h.vt != nil && !h.anyAttached() {
		h.vt.AnswerQueries(true)
	}
}

func (h *Holder) request(c *hclient, req proto.Request) {
	reply := func(v any, perr *proto.Error) {
		resp := proto.Response{ID: req.ID, OK: perr == nil, Error: perr}
		if v != nil {
			resp.Result, _ = json.Marshal(v)
		}
		_ = c.w.WriteJSON(proto.TResponse, resp)
	}
	switch req.Op {
	case "info":
		h.mu.Lock()
		m := *h.m
		h.mu.Unlock()
		reply(m, nil)
	case "peek":
		var args struct {
			Bytes int `json:"bytes"`
		}
		_ = json.Unmarshal(req.Args, &args)
		if args.Bytes <= 0 || args.Bytes > 1<<20 {
			args.Bytes = 256 << 10
		}
		h.mu.Lock()
		data, start := h.ring.Tail(args.Bytes)
		cols, rows := h.m.Term.Cols, h.m.Term.Rows
		h.mu.Unlock()
		reply(map[string]any{"data": data, "start": start, "cols": cols, "rows": rows}, nil)
	case "rename":
		var args struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(req.Args, &args) != nil || strings.TrimSpace(args.Name) == "" {
			reply(nil, &proto.Error{Code: "bad_request", Message: "a name is needed"})
			return
		}
		h.mu.Lock()
		h.m.Name = strings.TrimSpace(args.Name)
		h.dirty = true
		h.mu.Unlock()
		reply(map[string]string{"name": args.Name}, nil)
	case "ports":
		h.mu.Lock()
		unit, pid := h.m.Run.Unit != "", h.m.Run.AgentPID
		h.mu.Unlock()
		reply(listeningPorts(sessionPIDs(unit, pid)), nil)
	case "hook":
		var ev HookEvent
		if json.Unmarshal(req.Args, &ev) == nil {
			h.mu.Lock()
			h.applyHook(ev)
			h.mu.Unlock()
		}
		reply(map[string]bool{"ok": true}, nil)
	case "send":
		var args struct {
			Text  string `json:"text"`
			Enter bool   `json:"enter"`
		}
		if json.Unmarshal(req.Args, &args) != nil {
			reply(nil, &proto.Error{Code: "bad_request", Message: "text is needed"})
			return
		}
		h.mu.Lock()
		exited := h.exited
		h.needInput = false
		h.mu.Unlock()
		if exited {
			reply(nil, &proto.Error{Code: "exited", Message: "the session has ended"})
			return
		}
		data := []byte(args.Text)
		if args.Enter {
			data = append(data, '\r')
		}
		if _, err := h.ptmx.Write(data); err != nil {
			reply(nil, &proto.Error{Code: "write", Message: err.Error()})
			return
		}
		reply(map[string]int{"bytes": len(data)}, nil)
	case "kill":
		reply(map[string]bool{"killed": true}, nil)
		go h.stop("killed by request", false)
	default:
		reply(nil, &proto.Error{Code: "unknown_op", Message: "unknown request " + req.Op})
	}
}

// kill stops the agent's whole process group, then the holder.
func (h *Holder) kill(reason string) { h.stop(reason, false) }

// stop ends the agent's process group, then the holder. When system is true,
// the session is recorded as stopped with the host, not as ended, so restore
// brings it back after a restart.
func (h *Holder) stop(reason string, system bool) {
	h.mu.Lock()
	exited := h.exited
	h.stopping = system
	h.mu.Unlock()
	if !exited && h.cmd != nil && h.cmd.Process != nil {
		pid := h.cmd.Process.Pid
		_ = syscall.Kill(-pid, syscall.SIGHUP)
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			h.mu.Lock()
			exited = h.exited
			h.mu.Unlock()
			if exited {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if !exited {
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			time.Sleep(100 * time.Millisecond)
		}
	}
	if h.vt != nil {
		_ = h.vt.Close()
	}
	h.log.Printf("holder exiting: %s", reason)
	h.mu.Lock()
	switch {
	case system:
		h.m.Status.Reason, h.m.Status.Source = reason, "system"
	case h.m.Status.State != StateExited && h.m.Status.State != StateErrored:
		h.m.Status = Status{State: StateExited, Since: time.Now().UTC(), Reason: reason}
	}
	m := *h.m
	h.mu.Unlock()
	_ = m.Save()
	if h.ln != nil {
		h.ln.Close()
	}
	os.Remove(paths.Socket(h.id))
	os.Exit(0)
}

// selfUnit returns the systemd unit this process runs in, if any.
func selfUnit() string {
	b, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if rest, ok := strings.CutPrefix(line, "0::"); ok {
			if leaf := filepath.Base(rest); strings.HasSuffix(leaf, ".service") {
				return leaf
			}
		}
	}
	return ""
}

// readOOMKills reads the oom_kill counter of this process's cgroup.
func readOOMKills() int64 {
	b, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if rest, ok := strings.CutPrefix(line, "0::"); ok {
			ev, err := os.ReadFile(filepath.Join("/sys/fs/cgroup", rest, "memory.events"))
			if err != nil {
				return 0
			}
			for _, l := range strings.Split(string(ev), "\n") {
				if v, ok := strings.CutPrefix(l, "oom_kill "); ok {
					n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
					return n
				}
			}
		}
	}
	return 0
}

func envValue(env []string, key string) string {
	for i := len(env) - 1; i >= 0; i-- {
		if v, ok := strings.CutPrefix(env[i], key+"="); ok {
			return v
		}
	}
	return ""
}

// lookPath finds name in the given PATH value.
func lookPath(name, path string) (string, error) {
	if strings.Contains(name, "/") {
		return name, nil
	}
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s is not installed on this host (not found on PATH)", name)
}

// snapshotLines is how much scrollback a new client gets in a snapshot.
const snapshotLines = 5000

// startVT starts the shadow terminal that sits next to this binary. Without it
// the holder still works: new clients get the raw ring instead of a snapshot.
func (h *Holder) startVT() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	path := filepath.Join(filepath.Dir(exe), brand.VTBinary)
	if _, err := os.Stat(path); err != nil {
		h.log.Printf("no shadow terminal at %s; new clients get the raw output", path)
		return
	}
	h.ptyOut = make(chan []byte, 256)
	go func() {
		for b := range h.ptyOut {
			_, _ = h.ptmx.Write(b)
		}
	}()
	h.mu.Lock()
	cols, rows := h.m.Term.Cols, h.m.Term.Rows
	tp, tpv := envValue(h.m.Env, "TERM_PROGRAM"), envValue(h.m.Env, "TERM_PROGRAM_VERSION")
	h.mu.Unlock()
	vt, err := StartVT(VTOptions{
		Path: path, Cols: cols, Rows: rows,
		History: func() ([]byte, int64) {
			h.mu.Lock()
			defer h.mu.Unlock()
			d, _ := h.ring.ReadFrom(h.ring.Start())
			return d, h.ring.Start()
		},
		OnWrite: func(p []byte) {
			select {
			case h.ptyOut <- append([]byte(nil), p...):
			default: // the agent is not reading its input; drop the reply
			}
		},
		Logf: h.log.Printf,
	})
	if err != nil {
		h.log.Printf("shadow terminal did not start: %v", err)
		return
	}
	if tp != "" {
		vt.SetProfile(VTProfile{XTVersion: strings.TrimSpace(tp + " " + tpv)})
	}
	vt.AnswerQueries(true) // no client yet
	h.mu.Lock()
	h.vt = vt
	h.mu.Unlock()
}

// anyAttached reports whether a client is attached. The caller holds h.mu.
func (h *Holder) anyAttached() bool {
	for c := range h.clients {
		if c.attached {
			return true
		}
	}
	return false
}
