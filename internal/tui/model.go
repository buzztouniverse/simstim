package tui

import (
	"context"
	"slices"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/iAmLakshya/simstim/internal/client"
)

// screen is the one list the launcher shows (docs/ux.md 2): the hosts, or
// the sessions on one host.
type screen int

const (
	scrHosts screen = iota
	scrSessions
)

// sessKey names a session across hosts.
type sessKey struct{ host, id string }

// Options configure the launcher.
type Options struct {
	// Motion turns animation on. See motionOn.
	Motion bool
	// Now is the clock. Tests set it.
	Now func() time.Time
	// Switcher opens on the sessions of every host, for a quick switch.
	Switcher bool
}

// Model is the launcher: one list at a time, and the dialogs drawn over it.
type Model struct {
	ctx      context.Context
	be       Backend
	settings client.Settings
	now      func() time.Time
	motion   bool

	width, height int
	dark          bool
	richColor     bool // 256 colours or more: the brand gradient shows
	truecolor     bool
	colors        termColors // the terminal's own colours, for smooth blends
	focused       bool       // the terminal window has focus

	hosts    []*hostConn
	loadErr  error
	mem      memory   // what the last run saw (memory.go)
	sshHosts []string // the Host aliases in the user's SSH config

	screen     screen
	cur        string // the host the sessions screen shows; "" for every host
	hostSel    int    // index into visibleHosts()
	hostTop    int
	sel        sessKey
	want       sessKey // a session to select as soon as it is listed
	selIdx     int
	autoSel    bool // the selection was not picked by the user, so it stays on top
	sessTop    int
	hideExited bool
	prompt     *promptState // the search or quick-start line, when open

	preview previewState

	rename    *renameState
	modal     modal
	toasts    toastStack
	quitArmed bool

	// Motion: when it all began, when the list last appeared, when the
	// dialog on top opened; ticking is true while a frame tick is pending.
	epoch    time.Time
	revealAt time.Time
	modalAt  time.Time
	quitAt   time.Time
	slideAt  time.Time
	slideDir int // +1 into a host's sessions, -1 back to the hosts
	selAt    time.Time
	selFrom  int // the row the selection left, for its glide
	jacking  *jackIn
	ticking  bool

	traces     map[sessKey][]time.Time // moments of output, for each row's trace
	activity   map[sessKey]activity    // what each session is doing, from its screen
	activityAt time.Time               // when an activity line last changed
	geo        boxGeo                  // where the launcher's box was last drawn
	modalGeo   boxGeo                  // and the dialog on top

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

// renameState is the inline rename of a session row.
type renameState struct {
	key  sessKey
	old  string
	in   input
	busy bool
}

// New builds the launcher. It reads the saved hosts at once; everything
// that touches the network runs later, in commands.
func New(ctx context.Context, be Backend, o Options) *Model {
	m := &Model{
		ctx: ctx, be: be, now: o.Now, motion: o.Motion,
		width: 80, height: 24, dark: true, focused: true,
		autoSel:   true,
		attach:    &client.AttachState{},
		lastAgent: map[string]string{},
		traces:    map[sessKey][]time.Time{},
		activity:  map[sessKey]activity{},
		conns:     newConnSet(),
		exec:      tea.Exec,
		after: func(d time.Duration, msg tea.Msg) tea.Cmd {
			return tea.Tick(d, func(time.Time) tea.Msg { return msg })
		},
	}
	if m.now == nil {
		m.now = time.Now
	}
	m.epoch, m.revealAt = m.now(), m.now()
	m.settings = be.Settings()
	commands = withKeymap(registry(), m.settings.Keymap)
	hosts, err := be.LoadHosts()
	m.loadErr = err
	sort.SliceStable(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
	for _, h := range hosts {
		m.hosts = append(m.hosts, &hostConn{host: h})
	}
	m.restore(o.Switcher)
	if d, ok := be.(interface {
		demoTraces(time.Time) map[sessKey][]time.Time
	}); ok {
		m.traces = d.demoTraces(m.now())
	}
	switch {
	case o.Switcher && len(m.hosts) > 0:
		m.screen, m.cur = scrSessions, ""
	case m.want != (sessKey{}):
		// Where the last run left off.
	case len(m.hosts) == 1:
		// One host: its sessions are the only useful list.
		m.screen, m.cur = scrSessions, m.hosts[0].name()
	}
	m.syncSel()
	return m
}

// Close closes every connection and saves what the launcher saw. Run calls
// it after the program ends.
func (m *Model) Close() {
	m.remember()()
	m.conns.closeAll()
}

// Init starts the connections and the timers.
func (m *Model) Init() tea.Cmd {
	be := m.be
	cmds := []tea.Cmd{
		func() tea.Msg { return sshHostsMsg{be.SSHConfigHosts()} },
		tea.RequestBackgroundColor,
		tea.RequestForegroundColor,
		queryPalette(),
		m.after(refreshEvery, refreshTickMsg{}),
		m.after(latencyEvery, latencyTickMsg{}),
		m.after(activityEvery, activityTickMsg{}),
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
	previewTickMsg struct{ seq int }
	opDoneMsg      struct {
		host string
		ok   string
		err  error
		drop string // a session to take off the list
	}
	foldersMsg struct {
		host, prefix string
		res          client.FoldersResult
		quick        bool // for the quick-start line
	}
	activityTickMsg struct{}
	startedMsg      struct {
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
	before, modal, scr, cur, armed := m.sel, m.modal, m.screen, m.cur, m.quitArmed
	selIdx, hostSel := m.selIdx, m.hostSel
	cmd := m.update(msg)
	if m.sel != before {
		cmd = tea.Batch(cmd, m.schedulePreview())
	}
	// Moments that start an animation: a dialog opens, a list appears, the
	// quit chord arms.
	if m.modal != nil && m.modal != modal {
		m.modalAt = m.now()
	}
	if m.screen != scr || m.cur != cur {
		m.revealAt, m.slideAt, m.slideDir = m.now(), m.now(), 1
		if m.screen == scrHosts {
			m.slideDir = -1
		}
	} else if m.sel != before || m.hostSel != hostSel {
		m.selAt, m.selFrom = m.now(), selIdx
		if m.screen == scrHosts {
			m.selFrom = hostSel
		}
	}
	if m.quitArmed && !armed {
		m.quitAt = m.now()
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
		m.colors.bg = msg.Color
		return nil
	case tea.ForegroundColorMsg:
		m.colors.fg = msg.Color
		return nil
	case uv.UnknownOscEvent:
		m.colors.onOSC(string(msg))
		return nil
	case tea.ColorProfileMsg:
		m.truecolor = msg.Profile == colorprofile.TrueColor
		m.richColor = m.truecolor || msg.Profile == colorprofile.ANSI256
		return nil
	case tea.FocusMsg:
		m.focused = true
		return nil
	case tea.BlurMsg:
		m.focused = false
		return nil
	case animTickMsg:
		m.ticking = false
		return nil
	case jackMsg:
		return m.onJack()
	case activityMsg:
		m.onActivity(msg)
		return nil
	case activityTickMsg:
		return tea.Batch(m.after(activityEvery, activityTickMsg{}), m.readActivity())
	case tea.KeyPressMsg:
		return m.onKey(msg)
	case tea.MouseClickMsg:
		return m.onClick(msg)
	case tea.MouseWheelMsg:
		return m.onWheel(msg)
	case tea.PasteMsg:
		return m.onPaste(msg.Content)

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
	case shellDoneMsg:
		if msg.err != nil {
			return m.toastErr(&client.UserError{Msg: "the shell on " + msg.host + " ended with an error: " + msg.err.Error()})
		}
		return nil
	case foldersMsg:
		if p := m.prompt; msg.quick && p != nil && p.host == msg.host && p.asked == msg.prefix {
			p.folders = msg.res.Matches
		}
		if d, ok := m.modal.(*newSession); ok && d.host == msg.host && !msg.quick {
			d.setSuggestions(msg.prefix, msg.res)
		}
		return nil
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
		m.sshHosts = msg.hosts
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

// needSpin reports whether something on screen moves slowly: a spinner, a
// working glyph or a breathing dot.
func (m *Model) needSpin() bool {
	if b, ok := m.modal.(interface{ busy() bool }); ok && b.busy() {
		return true
	}
	for _, h := range m.hosts {
		if h.status == stConnecting {
			return true
		}
		for _, s := range h.sessions {
			if s.State == "working" || s.State == "starting" || s.State == "needs_input" {
				return true
			}
		}
	}
	return false
}

// Hosts and sessions as shown.

func (m *Model) visibleHosts() []*hostConn { return m.hosts }

// selectedHost is the host in play: the one picked in the hosts list, or
// the one whose sessions are shown. It is nil on the list of every host.
func (m *Model) selectedHost() *hostConn {
	if m.screen == scrSessions {
		return m.hostByName(m.cur)
	}
	vh := m.visibleHosts()
	if m.hostSel < 0 || m.hostSel >= len(vh) {
		return nil
	}
	return vh[m.hostSel]
}

type rowKind int

const (
	rowSession rowKind = iota
	rowHeader
	rowNote
)

// sessRow is one row of the sessions list.
type sessRow struct {
	kind rowKind
	h    *hostConn
	s    client.Session
}

func (r sessRow) key() sessKey { return sessKey{r.h.name(), r.s.ID} }

func (m *Model) sessionsOf(h *hostConn) []client.Session {
	var out []client.Session
	for _, s := range h.sessions {
		if m.hideExited && ended(s) {
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

// rows lists the sessions screen: one host's sessions, or every host's
// grouped under its name.
func (m *Model) rows() []sessRow {
	if m.screen != scrSessions {
		return nil
	}
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
		case h.status == stFailed:
			out = append(out, sessRow{kind: rowHeader, h: h}, sessRow{kind: rowNote, h: h})
		case h.status == stOffline:
			out = append(out, sessRow{kind: rowHeader, h: h})
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
	if m.screen != scrSessions || m.sel == (sessKey{}) {
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
// or on the nearest one if it went away. A wanted session (the one last
// attached to) is taken as soon as it is listed.
func (m *Model) syncSel() {
	m.hostSel = max(min(m.hostSel, len(m.visibleHosts())-1), 0)
	rows := m.rows()
	find := func(k sessKey) int {
		for i, r := range rows {
			if r.kind == rowSession && r.key() == k {
				return i
			}
		}
		return -1
	}
	idx := -1
	if m.want != (sessKey{}) {
		if idx = find(m.want); idx >= 0 {
			m.want, m.autoSel = sessKey{}, false
		}
	}
	if idx < 0 {
		idx = find(m.sel)
		switch {
		case m.autoSel:
			idx = nearestSession(rows, 0)
		case idx < 0:
			idx = nearestSession(rows, m.selIdx)
		}
	}
	m.selIdx = max(idx, 0)
	if idx >= 0 {
		m.sel = rows[idx].key()
	} else {
		m.sel = sessKey{}
	}
	m.scrollTo()
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

// scrollTo keeps the selected row in view.
func (m *Model) scrollTo() {
	listH := m.listRoom()
	if m.screen == scrHosts {
		n := len(m.visibleHosts())
		if m.hostSel < m.hostTop {
			m.hostTop = m.hostSel
		}
		if m.hostSel >= m.hostTop+listH {
			m.hostTop = m.hostSel - listH + 1
		}
		m.hostTop = max(0, min(m.hostTop, n-listH))
		return
	}
	n := len(m.rows())
	if m.selIdx < m.sessTop {
		m.sessTop = m.selIdx
	}
	if m.selIdx >= m.sessTop+listH {
		m.sessTop = m.selIdx - listH + 1
	}
	m.sessTop = max(0, min(m.sessTop, n-listH))
	// A group header just above the first row stays in view.
	if m.sessTop == 1 && m.selIdx == 1 {
		m.sessTop = 0
	}
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
	m.selIdx, m.sel, m.autoSel, m.want = i, rows[i].key(), false, sessKey{}
	m.scrollTo()
}

func (m *Model) jumpSel(d int) {
	rows := m.rows()
	target := min(max(m.selIdx+d, 0), len(rows)-1)
	if i := nearestSession(rows, target); i >= 0 {
		m.selIdx, m.sel, m.autoSel = i, rows[i].key(), false
		m.scrollTo()
	}
}

func (m *Model) moveHost(d int) {
	n := len(m.visibleHosts())
	m.hostSel = min(max(m.hostSel+d, 0), max(n-1, 0))
	m.scrollTo()
}

// openHost shows the sessions of h, or of every host for nil.
func (m *Model) openHost(h *hostConn) {
	m.screen, m.cur = scrSessions, ""
	if h != nil {
		m.cur = h.name()
	}
	m.prompt = nil
	m.selIdx, m.sessTop, m.autoSel = 0, 0, true
	m.sel = sessKey{}
	m.syncSel()
}

// backToHosts returns to the hosts list, on the host that was open.
func (m *Model) backToHosts() {
	cur := m.cur
	m.screen, m.sel, m.rename, m.prompt = scrHosts, sessKey{}, nil, nil
	for i, h := range m.visibleHosts() {
		if h.name() == cur {
			m.hostSel = i
		}
	}
	m.syncSel()
}
