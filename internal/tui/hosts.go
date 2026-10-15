package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/iamlakshya/simstim/internal/api"
	"github.com/iamlakshya/simstim/internal/brand"
	"github.com/iamlakshya/simstim/internal/client"
)

// connStatus is where a host's background connection stands.
type connStatus int

const (
	stConnecting connStatus = iota
	stOnline
	stOffline // unreachable; retrying with backoff
	stFailed  // reachable, but something needs fixing; retrying slowly
)

// hostConn is one saved host and its background connection. Only Update
// touches it; the work happens in tea.Cmds that report back with messages.
type hostConn struct {
	host     client.Host
	status   connStatus
	conn     Conn
	gen      int // bumped on every dial, so late answers from old connections are dropped
	err      error
	sessions []client.Session
	loaded   bool
	latency  time.Duration
	info     *client.HostInfo
	backoff  time.Duration
	listing  bool
	pinging  bool
	// changes counts local edits to the list (a new, renamed or deleted
	// session). A list asked for before the latest edit is stale.
	changes int
}

func (h *hostConn) name() string { return h.host.Name }

func (h *hostConn) online() bool { return h.status == stOnline && h.conn != nil }

// Messages from the connection commands.
type (
	dialledMsg struct {
		name string
		gen  int
		conn Conn
		err  error
	}
	connDoneMsg struct {
		name string
		gen  int
	}
	redialMsg struct {
		name string
		gen  int
	}
	sessionsMsg struct {
		name    string
		gen     int
		changes int
		list    []client.Session
		err     error
	}
	latencyMsg struct {
		name string
		gen  int
		d    time.Duration
	}
	doctorMsg struct {
		name string
		gen  int
		info client.HostInfo
		err  error
	}
	refreshTickMsg struct{}
	latencyTickMsg struct{}
)

func (m *Model) hostByName(name string) *hostConn {
	for _, h := range m.hosts {
		if h.name() == name {
			return h
		}
	}
	return nil
}

// dial starts a background connection to h.
func (m *Model) dial(h *hostConn) tea.Cmd {
	h.gen++
	h.status = stConnecting
	h.listing, h.pinging = false, false
	gen, host, be, ctx, set := h.gen, h.host, m.be, m.ctx, m.conns
	return func() tea.Msg {
		dctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		c, err := be.Dial(dctx, host)
		if err == nil && !set.add(c) {
			return nil
		}
		return dialledMsg{name: host.Name, gen: gen, conn: c, err: err}
	}
}

func waitDone(name string, gen int, c Conn) tea.Cmd {
	return func() tea.Msg {
		<-c.Done()
		return connDoneMsg{name: name, gen: gen}
	}
}

func (m *Model) dropConn(c Conn) tea.Cmd {
	set := m.conns
	return func() tea.Msg {
		set.drop(c)
		return nil
	}
}

// listSessions asks h for its sessions.
func (m *Model) listSessions(h *hostConn) tea.Cmd {
	if !h.online() || h.listing {
		return nil
	}
	h.listing = true
	name, gen, changes, c, ctx := h.name(), h.gen, h.changes, h.conn, m.ctx
	return func() tea.Msg {
		cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		list, err := c.Sessions(cctx)
		return sessionsMsg{name: name, gen: gen, changes: changes, list: list, err: err}
	}
}

func (m *Model) ping(h *hostConn) tea.Cmd {
	if !h.online() || h.pinging {
		return nil
	}
	h.pinging = true
	name, gen, c, ctx := h.name(), h.gen, h.conn, m.ctx
	return func() tea.Msg {
		cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		d, err := c.RoundTrip(cctx)
		if err != nil {
			d = 0
		}
		return latencyMsg{name: name, gen: gen, d: d}
	}
}

func (m *Model) doctor(h *hostConn) tea.Cmd {
	name, gen, c, ctx := h.name(), h.gen, h.conn, m.ctx
	return func() tea.Msg {
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		info, err := c.Doctor(cctx)
		return doctorMsg{name: name, gen: gen, info: info, err: err}
	}
}

// nextBackoff is 1 s, then doubles up to 30 s.
func nextBackoff(d time.Duration) time.Duration {
	if d <= 0 {
		return time.Second
	}
	return min(d*2, maxBackoff)
}

// offline reports whether err means the host could not be reached at all,
// as opposed to reached but not ready.
func offline(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, client.ErrClosed) {
		return true
	}
	msg, _ := errText(err)
	return strings.HasPrefix(msg, "cannot reach") || strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "Connection refused") || strings.Contains(msg, "No route to host")
}

// onConn handles the connection messages.
func (m *Model) onConn(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case dialledMsg:
		return m.onDialled(msg)
	case connDoneMsg:
		return m.onConnDone(msg)
	case redialMsg:
		return m.onRedial(msg)
	case sessionsMsg:
		return m.onSessions(msg)
	case latencyMsg:
		return m.onLatency(msg)
	case doctorMsg:
		return m.onDoctor(msg)
	case refreshTickMsg:
		return m.onRefreshTick()
	case latencyTickMsg:
		return m.onLatencyTick()
	}
	return nil
}

// onDialled takes a finished dial: a live connection, or an error and a retry later.
func (m *Model) onDialled(msg dialledMsg) tea.Cmd {
	h := m.hostByName(msg.name)
	if h == nil || h.gen != msg.gen {
		if msg.conn != nil {
			return m.dropConn(msg.conn)
		}
		return nil
	}
	if msg.err != nil {
		h.err = msg.err
		h.status = stFailed
		if offline(msg.err) {
			h.status = stOffline
		}
		h.backoff = nextBackoff(h.backoff)
		return m.after(h.backoff, redialMsg{name: h.name(), gen: h.gen})
	}
	h.conn, h.status, h.err, h.backoff = msg.conn, stOnline, nil, 0
	cmds := []tea.Cmd{m.listSessions(h), m.ping(h), m.doctor(h), waitDone(h.name(), h.gen, h.conn)}
	if m.previewOpen() && m.preview.key.host == h.name() {
		cmds = append(cmds, m.peek())
	}
	return tea.Batch(cmds...)
}

// onConnDone notes a lost connection and dials again later.
func (m *Model) onConnDone(msg connDoneMsg) tea.Cmd {
	h := m.hostByName(msg.name)
	if h == nil || h.gen != msg.gen || h.conn == nil {
		return nil
	}
	old := h.conn
	h.conn, h.status, h.latency = nil, stOffline, 0
	h.err = &client.UserError{
		Msg:  "lost the connection to " + h.name(),
		Hint: fmt.Sprintf("%s host test %s", brand.Name, h.name()),
	}
	h.backoff = nextBackoff(h.backoff)
	return tea.Batch(m.dropConn(old), m.after(h.backoff, redialMsg{name: h.name(), gen: h.gen}))
}

// onRedial dials a host again, unless it is gone or back online.
func (m *Model) onRedial(msg redialMsg) tea.Cmd {
	h := m.hostByName(msg.name)
	if h == nil || h.gen != msg.gen || h.online() {
		return nil
	}
	return m.dial(h)
}

// onSessions takes a host's list of sessions.
func (m *Model) onSessions(msg sessionsMsg) tea.Cmd {
	h := m.hostByName(msg.name)
	if h == nil || h.gen != msg.gen {
		return nil
	}
	h.listing = false
	if msg.changes != h.changes {
		return m.listSessions(h) // asked for before a local edit: ask again
	}
	if msg.err != nil {
		if !errors.Is(msg.err, client.ErrClosed) && !errors.Is(msg.err, context.Canceled) {
			h.err = msg.err
		}
		return nil
	}
	h.err = nil
	cmd := m.alertNew(h, msg.list)
	m.noteActivity(h, msg.list)
	h.sessions, h.loaded = msg.list, true
	m.syncSel()
	if m.want.host == h.name() {
		m.want = sessKey{} // it is gone: the first live list did not have it
	}
	return cmd
}

// onLatency takes a ping's round trip.
func (m *Model) onLatency(msg latencyMsg) tea.Cmd {
	h := m.hostByName(msg.name)
	if h == nil || h.gen != msg.gen {
		return nil
	}
	h.pinging = false
	if msg.d > 0 {
		h.latency = msg.d
	}
	return nil
}

// onDoctor keeps a host's facts, for the details and the new-session form.
func (m *Model) onDoctor(msg doctorMsg) tea.Cmd {
	h := m.hostByName(msg.name)
	if h == nil || h.gen != msg.gen || msg.err != nil {
		return nil
	}
	info := msg.info
	h.info = &info
	if d, ok := m.modal.(*newSession); ok {
		d.refresh(m)
	}
	return nil
}

// onRefreshTick lists every host's sessions again, and refreshes a peek.
func (m *Model) onRefreshTick() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2+len(m.hosts))
	cmds = append(cmds, m.after(refreshEvery, refreshTickMsg{}))
	for _, h := range m.hosts {
		cmds = append(cmds, m.listSessions(h))
	}
	if s := m.selected(); s != nil && s.Alive && m.previewOpen() {
		cmds = append(cmds, m.peek())
	}
	return tea.Batch(cmds...)
}

// onLatencyTick pings every host.
func (m *Model) onLatencyTick() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2+len(m.hosts))
	cmds = append(cmds, m.after(latencyEvery, latencyTickMsg{}))
	for _, h := range m.hosts {
		cmds = append(cmds, m.ping(h))
	}
	return tea.Batch(cmds...)
}

// reconnect drops h's connection, if any, and dials again now.
func (m *Model) reconnect(h *hostConn) tea.Cmd {
	var cmds []tea.Cmd
	if h.conn != nil {
		cmds = append(cmds, m.dropConn(h.conn))
		h.conn = nil
	}
	h.backoff = 0
	return tea.Batch(append(cmds, m.dial(h))...)
}

// addHost puts a new or changed host in the list. If conn is set, it is a
// ready connection to adopt, so the host needs no second dial.
func (m *Model) addHost(host client.Host, conn Conn, info *client.HostInfo) tea.Cmd {
	h := m.hostByName(host.Name)
	var cmds []tea.Cmd
	if h == nil {
		h = &hostConn{host: host}
		m.hosts = append(m.hosts, h)
		sort.SliceStable(m.hosts, func(i, j int) bool { return m.hosts[i].name() < m.hosts[j].name() })
	} else {
		h.host = host
		if h.conn != nil {
			cmds = append(cmds, m.dropConn(h.conn))
			h.conn = nil
		}
	}
	if conn == nil {
		return tea.Batch(append(cmds, m.dial(h))...)
	}
	h.gen++
	h.conn, h.status, h.err, h.backoff, h.info = conn, stOnline, nil, 0, info
	h.listing, h.pinging = false, false
	return tea.Batch(append(cmds, m.listSessions(h), m.ping(h), waitDone(h.name(), h.gen, conn))...)
}

// removeHost forgets h here. Its sessions keep running on the host.
func (m *Model) removeHost(name string) tea.Cmd {
	for i, h := range m.hosts {
		if h.name() != name {
			continue
		}
		m.hosts = append(m.hosts[:i], m.hosts[i+1:]...)
		h.gen++
		if m.cur == name {
			m.screen, m.cur = scrHosts, ""
		}
		m.syncSel()
		if h.conn != nil {
			return m.dropConn(h.conn)
		}
		return nil
	}
	return nil
}

// Session order and looks.

// rank orders sessions: needs input, then working, idle, exited.
func rank(s client.Session) int {
	switch s.State {
	case api.StateNeedsInput:
		return 0
	case api.StateWorking, api.StateStarting:
		return 1
	case api.StateIdle:
		return 2
	default:
		return 3
	}
}

func sortSessions(list []client.Session) {
	sort.SliceStable(list, func(i, j int) bool {
		ri, rj := rank(list[i]), rank(list[j])
		if ri != rj {
			return ri < rj
		}
		if !list[i].CreatedAt.Equal(list[j].CreatedAt) {
			return list[i].CreatedAt.After(list[j].CreatedAt)
		}
		return list[i].ID < list[j].ID
	})
}

func ended(s client.Session) bool { return rank(s) == 3 }

// look is how a session's state is drawn: its glyph and its label. A
// working session's glyph is alive; one that needs the user breathes.
func (m *Model) look(s client.Session) (glyph, label seg) {
	switch s.State {
	case api.StateNeedsInput:
		return seg{glyphNeedsInput, m.breath(sWarning)}, seg{"needs input", sWarning}
	case api.StateWorking:
		return seg{m.working(s.ID), sKey}, seg{"working", sKey}
	case api.StateStarting:
		return seg{m.spinner(), sKey}, seg{"starting", sFaint}
	case api.StateIdle:
		return seg{glyphIdle, sFaint}, seg{"idle", sFaint}
	case api.StateExited:
		switch {
		case s.Signal != "":
			return seg{glyphFail, sDanger}, seg{"stopped " + strings.TrimPrefix(s.Signal, "SIG"), sDanger}
		case s.ExitCode != nil && *s.ExitCode != 0:
			return seg{glyphFail, sDanger}, seg{fmt.Sprintf("exited %d", *s.ExitCode), sDanger}
		default:
			return seg{glyphOK, sSuccess}, seg{"exited 0", sFaint}
		}
	case api.StateErrored:
		if s.OOM {
			return seg{glyphFail, sDanger}, seg{"out of memory", sDanger}
		}
		return seg{glyphFail, sDanger}, seg{"failed", sDanger}
	case api.StateLost:
		return seg{glyphOffline, sFaint}, seg{"lost", sFaint}
	}
	return seg{glyphIdle, sFaint}, seg{strings.ReplaceAll(s.State, "_", " "), sFaint}
}

// homePath shows a path on h with its home folder as "~".
func homePath(h *hostConn, p string) string {
	if h == nil || h.info == nil || h.info.User == "" {
		return p
	}
	home := "/home/" + h.info.User
	if h.info.User == "root" {
		home = "/root"
	}
	if p == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(p, home+"/"); ok {
		return "~/" + rest
	}
	return p
}

// since is when the session entered its state, for the age column.
func since(s client.Session) time.Time {
	if !s.Since.IsZero() {
		return s.Since
	}
	return s.CreatedAt
}

// alertNew raises a desktop notification (OSC 777) when a session starts to
// need the user while the terminal is in the background. The launcher draws
// the same news in its list; this is for when nobody is looking.
func (m *Model) alertNew(h *hostConn, list []client.Session) tea.Cmd {
	if m.focused || !h.loaded {
		return nil
	}
	before := map[string]string{}
	for _, s := range h.sessions {
		before[s.ID] = s.State
	}
	var b strings.Builder
	for _, s := range list {
		if s.State != api.StateNeedsInput || before[s.ID] == api.StateNeedsInput {
			continue
		}
		body := s.Name + " needs you"
		if s.Reason != "" {
			body += ": " + s.Reason
		}
		b.WriteString("\x1b]777;notify;" + oscText(brand.DisplayName+" · "+h.name()) + ";" + oscText(body) + "\x1b\\")
	}
	if b.Len() == 0 {
		return nil
	}
	return tea.Raw(b.String())
}

// oscText keeps text safe inside an OSC string.
func oscText(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == ';' {
			return ' '
		}
		return r
	}, s)
}
