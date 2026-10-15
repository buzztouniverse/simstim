package tui

import (
	"context"
	"slices"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/iAmLakshya/simstim/internal/client"
)

type pane int

const (
	paneHosts pane = iota
	paneSessions
)

// sessKey names a session across hosts.
type sessKey struct{ host, id string }

// Options configure the home screen.
type Options struct {
	// Motion turns animation on. See motionOn.
	Motion bool
	// Now is the clock. Tests set it.
	Now func() time.Time
}

// Model is the home screen and everything drawn over it.
type Model struct {
	ctx      context.Context
	be       Backend
	settings client.Settings
	now      func() time.Time
	motion   bool

	width, height int
	dark          bool
	truecolor     bool

	hosts   []*hostConn
	loadErr error

	focus      pane
	hostSel    int // 0 is "all hosts"; i+1 is visibleHosts()[i]
	sel        sessKey
	selIdx     int
	autoSel    bool // the selection was not picked by the user, so it stays on top
	sessTop    int
	hostTop    int
	hostFilter filter
	sessFilter filter
	hideExited bool

	showPreview bool
	preview     previewState

	rename *renameState
	modal  modal
	toasts toastStack

	spin      int
	spinning  bool
	hero      int
	heroOn    bool
	quitArmed time.Time

	// attach is kept for the whole run, so attaching to the same session
	// again continues from the byte the terminal already shows.
	attach    *client.AttachState
	lastAgent map[string]string
	conns     *connSet

	// exec hands the terminal to a blocking command; after delivers msg
	// after d. Tests replace both.
	exec  func(tea.ExecCommand, tea.ExecCallback) tea.Cmd
	after func(d time.Duration, msg tea.Msg) tea.Cmd
}

// filter is a filter-as-you-type over a list.
type filter struct {
	typing bool
	in     input
}

func (f *filter) query() string { return strings.TrimSpace(f.in.value()) }

// renameState is the inline rename of a session row.
type renameState struct {
	key  sessKey
	old  string
	in   input
	busy bool
}

// New builds the home screen. It reads the saved hosts at once; everything
// that touches the network runs later, in commands.
func New(ctx context.Context, be Backend, o Options) *Model {
	m := &Model{
		ctx: ctx, be: be, now: o.Now, motion: o.Motion,
		width: 80, height: 24, dark: true,
		showPreview: true, autoSel: true,
		attach:    &client.AttachState{},
		lastAgent: map[string]string{},
		conns:     newConnSet(),
		exec:      tea.Exec,
		after: func(d time.Duration, msg tea.Msg) tea.Cmd {
			return tea.Tick(d, func(time.Time) tea.Msg { return msg })
		},
	}
	if m.now == nil {
		m.now = time.Now
	}
	m.settings = be.Settings()
	commands = withKeymap(registry(), m.settings.Keymap)
	hosts, err := be.LoadHosts()
	m.loadErr = err
	sort.SliceStable(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
	for _, h := range hosts {
		m.hosts = append(m.hosts, &hostConn{host: h})
	}
	if len(m.hosts) == 1 {
		m.hostSel = 1
	}
	return m
}

// Close closes every connection. Run calls it after the program ends.
func (m *Model) Close() { m.conns.closeAll() }

// Init starts the connections and the timers.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.RequestBackgroundColor,
		m.after(refreshEvery, refreshTickMsg{}),
		m.after(latencyEvery, latencyTickMsg{}),
	}
	for _, h := range m.hosts {
		cmds = append(cmds, m.dial(h))
	}
	if m.loadErr != nil {
		msg, hint := errText(m.loadErr)
		m.toasts.push(m.now(), toastError, sentence(msg), hint)
		cmds = append(cmds, m.after(toastErrorLife, toastTickMsg{}))
	}
	return tea.Batch(append(cmds, m.startTimers())...)
}

// Timer and result messages.
type (
	spinTickMsg    struct{}
	heroTickMsg    struct{}
	previewTickMsg struct{ seq int }
	opDoneMsg      struct {
		host string
		ok   string
		err  error
		drop string // a session to take off the list
	}
	startedMsg struct {
		host   string
		s      *client.Session
		err    error
		attach bool
	}
	attachDoneMsg struct {
		host, id, name string
		res            client.AttachResult
		err            error
	}
	hostSavedMsg struct {
		host   client.Host
		remove bool
		err    error
	}
	settingsMsg struct{ s client.Settings }
	sshHostsMsg struct{ hosts []string }
)

// Update handles one message. It never blocks: everything slow is a command.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	before := m.sel
	cmd := m.update(msg)
	if m.sel != before {
		cmd = tea.Batch(cmd, m.schedulePreview())
	}
	return m, tea.Batch(cmd, m.startTimers())
}

func (m *Model) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.syncSel()
		return nil
	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		return nil
	case tea.ColorProfileMsg:
		m.truecolor = msg.Profile == colorprofile.TrueColor
		return nil
	case tea.KeyPressMsg:
		return m.onKey(msg)
	case tea.PasteMsg:
		return m.onPaste(msg.Content)

	case spinTickMsg:
		m.spin++
		m.spinning = false
		return nil
	case heroTickMsg:
		m.hero++
		m.heroOn = false
		return nil
	case toastTickMsg:
		if m.toasts.expire(m.now()) {
			if d, ok := m.toasts.next(m.now()); ok {
				return m.after(d, toastTickMsg{})
			}
		}
		return nil

	case dialledMsg, connDoneMsg, redialMsg, sessionsMsg, latencyMsg, doctorMsg, refreshTickMsg, latencyTickMsg:
		return m.onConn(msg)

	case previewTickMsg:
		if msg.seq == m.preview.seq {
			return m.peek()
		}
		return nil
	case peekMsg:
		m.onPeek(msg)
		return nil

	case opDoneMsg:
		var cmd tea.Cmd
		if msg.err != nil {
			cmd = m.toastErr(msg.err)
		} else if msg.ok != "" {
			cmd = m.toast(toastSuccess, msg.ok)
		}
		if h := m.hostByName(msg.host); h != nil {
			h.changes++
			if msg.err == nil && msg.drop != "" {
				h.sessions = slices.DeleteFunc(h.sessions, func(s client.Session) bool { return s.ID == msg.drop })
				m.syncSel()
			}
			cmd = tea.Batch(cmd, m.listSessions(h))
		}
		if m.rename != nil && m.rename.busy {
			m.rename = nil
		}
		return cmd
	case startedMsg:
		return m.onStarted(msg)
	case attachDoneMsg:
		return m.onAttachDone(msg)
	case hostSavedMsg:
		return m.onHostSaved(msg)
	case setupMsg:
		if w, ok := m.modal.(*wizard); ok {
			return w.onMsg(m, msg)
		}
		if msg.conn != nil {
			return m.dropConn(msg.conn) // the wizard closed while it ran
		}
		return nil
	case editorDoneMsg:
		be := m.be
		reload := func() tea.Msg { return settingsMsg{be.Settings()} }
		if msg.err != nil {
			return tea.Batch(reload, m.toastErr(msg.err))
		}
		return reload
	case settingsMsg:
		m.settings = msg.s
		commands = withKeymap(registry(), m.settings.Keymap)
		return nil
	case sshHostsMsg:
		if w, ok := m.modal.(*wizard); ok {
			w.known = msg.hosts
		}
		return nil
	}
	if w, ok := m.modal.(*wizard); ok {
		return w.onMsg(m, msg)
	}
	return nil
}

// startTimers starts the spinner and hero ticks when something needs them.
// Idle screens never repaint.
func (m *Model) startTimers() tea.Cmd {
	if !m.motion {
		return nil
	}
	var cmds []tea.Cmd
	if !m.spinning && m.needSpin() {
		m.spinning = true
		cmds = append(cmds, m.after(spinEvery, spinTickMsg{}))
	}
	if !m.heroOn && len(m.hosts) == 0 && m.modal == nil {
		m.heroOn = true
		d := heroEvery
		if m.hero%heroCycle >= heroPulse {
			// Rest between pulses: one repaint, then nothing for a while.
			d = heroEvery * time.Duration(heroCycle-m.hero%heroCycle)
			m.hero += heroCycle - m.hero%heroCycle - 1
		}
		cmds = append(cmds, m.after(d, heroTickMsg{}))
	}
	return tea.Batch(cmds...)
}

func (m *Model) needSpin() bool {
	if b, ok := m.modal.(interface{ busy() bool }); ok && b.busy() {
		return true
	}
	for _, h := range m.hosts {
		if h.status == stConnecting {
			return true
		}
		for _, s := range h.sessions {
			if s.State == "working" || s.State == "starting" {
				return true
			}
		}
	}
	return false
}

// spinner is the current spinner frame, or a still one with motion off.
func (m *Model) spinner() string {
	if !m.motion {
		return glyphSpinStill
	}
	return spinFrames[m.spin%len(spinFrames)]
}

// Hosts and sessions as shown.

func (m *Model) visibleHosts() []*hostConn {
	q := m.hostFilter.query()
	if q == "" {
		return m.hosts
	}
	var out []*hostConn
	for _, h := range m.hosts {
		if _, ok := fuzzy(q, h.name()); ok {
			out = append(out, h)
		}
	}
	return out
}

// selectedHost is the host picked in the hosts pane, or nil for all hosts.
func (m *Model) selectedHost() *hostConn {
	vh := m.visibleHosts()
	if m.hostSel <= 0 || m.hostSel > len(vh) {
		return nil
	}
	return vh[m.hostSel-1]
}

type rowKind int

const (
	rowSession rowKind = iota
	rowHeader
	rowNote
)

// sessRow is one row of the sessions pane.
type sessRow struct {
	kind rowKind
	h    *hostConn
	s    client.Session
}

func (r sessRow) key() sessKey { return sessKey{r.h.name(), r.s.ID} }

func (m *Model) sessionsOf(h *hostConn) []client.Session {
	q := m.sessFilter.query()
	var out []client.Session
	for _, s := range h.sessions {
		if m.hideExited && ended(s) {
			continue
		}
		if q != "" && !matchSession(q, s) {
			continue
		}
		out = append(out, s)
	}
	sortSessions(out)
	return out
}

func matchSession(q string, s client.Session) bool {
	if _, ok := fuzzy(q, s.Name); ok {
		return true
	}
	lq := strings.ToLower(q)
	return strings.HasPrefix(s.ID, q) || strings.Contains(strings.ToLower(s.Agent), lq) || strings.Contains(strings.ToLower(s.Cwd), lq)
}

// rows lists the sessions pane: one host's sessions, or every host's grouped
// under a header.
func (m *Model) rows() []sessRow {
	if h := m.selectedHost(); h != nil {
		var out []sessRow
		for _, s := range m.sessionsOf(h) {
			out = append(out, sessRow{kind: rowSession, h: h, s: s})
		}
		return out
	}
	var out []sessRow
	for _, h := range m.hosts {
		list := m.sessionsOf(h)
		switch {
		case len(list) > 0:
			out = append(out, sessRow{kind: rowHeader, h: h})
			for _, s := range list {
				out = append(out, sessRow{kind: rowSession, h: h, s: s})
			}
		case !h.online() && h.status != stConnecting && m.sessFilter.query() == "":
			out = append(out, sessRow{kind: rowHeader, h: h}, sessRow{kind: rowNote, h: h})
		}
	}
	return out
}

func sessionRows(rows []sessRow) int {
	n := 0
	for _, r := range rows {
		if r.kind == rowSession {
			n++
		}
	}
	return n
}

// selected is the selected session, or nil.
func (m *Model) selected() *client.Session {
	h, s := m.selectedPair()
	if h == nil {
		return nil
	}
	return s
}

func (m *Model) selectedPair() (*hostConn, *client.Session) {
	if m.sel == (sessKey{}) {
		return nil, nil
	}
	h := m.hostByName(m.sel.host)
	if h == nil {
		return nil, nil
	}
	for i := range h.sessions {
		if h.sessions[i].ID == m.sel.id {
			return h, &h.sessions[i]
		}
	}
	return nil, nil
}

// syncSel keeps the selection on the same session after the list changes,
// or on the nearest one if it went away.
func (m *Model) syncSel() {
	rows := m.rows()
	idx := -1
	for i, r := range rows {
		if r.kind == rowSession && r.key() == m.sel {
			idx = i
			break
		}
	}
	switch {
	case m.autoSel:
		idx = nearestSession(rows, 0)
	case idx < 0:
		idx = nearestSession(rows, m.selIdx)
	}
	m.selIdx = max(idx, 0)
	if idx >= 0 {
		m.sel = rows[idx].key()
	} else {
		m.sel = sessKey{}
	}
	m.scrollTo(len(rows))
}

func nearestSession(rows []sessRow, i int) int {
	if len(rows) == 0 {
		return -1
	}
	i = min(max(i, 0), len(rows)-1)
	for j := i; j < len(rows); j++ {
		if rows[j].kind == rowSession {
			return j
		}
	}
	for j := i; j >= 0; j-- {
		if rows[j].kind == rowSession {
			return j
		}
	}
	return -1
}

// scrollTo keeps the selected row and the selected host in view.
func (m *Model) scrollTo(n int) {
	l := m.layout()
	if m.selIdx < m.sessTop {
		m.sessTop = m.selIdx
	}
	if m.selIdx >= m.sessTop+l.listH {
		m.sessTop = m.selIdx - l.listH + 1
	}
	m.sessTop = max(0, min(m.sessTop, n-l.listH))
	// A group header just above the first row stays in view.
	if m.sessTop == 1 && m.selIdx == 1 {
		m.sessTop = 0
	}
	hostRows := l.bodyH - 1
	if m.hostSel < m.hostTop {
		m.hostTop = m.hostSel
	}
	if m.hostSel >= m.hostTop+hostRows {
		m.hostTop = m.hostSel - hostRows + 1
	}
	m.hostTop = max(0, m.hostTop)
}

// moveSel moves the session selection by d rows, skipping headers.
func (m *Model) moveSel(d int) {
	rows := m.rows()
	if len(rows) == 0 {
		return
	}
	i := m.selIdx
	for {
		i += d
		if i < 0 || i >= len(rows) {
			return
		}
		if rows[i].kind == rowSession {
			break
		}
	}
	m.selIdx, m.sel, m.autoSel = i, rows[i].key(), false
	m.scrollTo(len(rows))
}

func (m *Model) moveHost(d int) {
	n := len(m.visibleHosts())
	m.hostSel = min(max(m.hostSel+d, 0), n)
	m.selIdx, m.sessTop, m.autoSel = 0, 0, true
	m.sel = sessKey{}
	m.syncSel()
}
