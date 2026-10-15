package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
)

// The add-host wizard (docs/contributing/ux-spec.md 2.2): one step per screen, Esc goes back.
type wizStep int

const (
	wzWhere wizStep = iota
	wzTest
	wzInstall
	wzReady
)

const wizSteps = 4

type checkState int

const (
	ckRunning checkState = iota
	ckOK
	ckFailed
	ckNote
)

// check is one line of a step's progress: a spinner, then ✓ or ✕.
type check struct {
	name   string
	state  checkState
	detail string
	took   time.Duration
}

type wizard struct {
	step  wizStep
	field int // 0 target, 1 name, 2 SSH config
	in    [3]input
	named bool // the name was typed, so it no longer follows the target
	known []string
	pick  int

	host   client.Host
	plat   client.Platform
	checks []check
	err    error
	run    bool
	seq    int
	cancel context.CancelFunc
	conn   Conn
	info   *client.HostInfo
	saved  bool
}

// Wizard results.
type (
	probeMsg struct {
		seq  int
		p    client.Platform
		err  error
		took time.Duration
	}
	installMsg struct {
		seq  int
		err  error
		took time.Duration
	}
	setupMsg struct {
		seq  int
		conn Conn
		res  client.SetupResult
		info client.HostInfo
		err  error
		took time.Duration
	}
)

func (m *Model) openWizard() tea.Cmd {
	w := &wizard{pick: -1}
	w.in[0] = newInput("", "a Host from your SSH config, or user@host")
	w.in[1] = newInput("", "the SSH target")
	w.in[2] = newInput("", "optional: an extra SSH config file")
	m.modal = w
	be := m.be
	return func() tea.Msg { return sshHostsMsg{be.SSHConfigHosts()} }
}

func (w *wizard) busy() bool { return w.run }

func (w *wizard) paste(s string) {
	if w.step == wzWhere {
		w.in[w.field].insert(s)
		w.follow()
	}
}

// follow keeps the name in step with the target until it is typed.
func (w *wizard) follow() {
	if w.named {
		return
	}
	t := w.in[0].value()
	if i := strings.LastIndex(t, "@"); i >= 0 {
		t = t[i+1:]
	}
	if i := strings.Index(t, ":"); i >= 0 {
		t = t[:i]
	}
	w.in[1].set(t)
}

// matches are the SSH config hosts that fit what is typed.
func (w *wizard) matches() []string {
	q := w.in[0].value()
	if q == "" || w.pick >= 0 {
		return w.known
	}
	var out []string
	for _, k := range w.known {
		if _, ok := fuzzy(q, k); ok {
			out = append(out, k)
		}
	}
	return out
}

func (w *wizard) stop() {
	w.seq++
	w.run = false
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
}

func (w *wizard) key(m *Model, k tea.KeyPressMsg) tea.Cmd {
	ks := k.String()
	if ks == "esc" {
		return w.back(m)
	}
	switch w.step {
	case wzWhere:
		return w.whereKey(m, k)
	case wzTest:
		if ks == "enter" && !w.run {
			if w.err != nil {
				return w.probe(m)
			}
			if w.plat.Installed {
				return w.setup(m)
			}
			w.step, w.err = wzInstall, nil
		}
	case wzInstall:
		if ks == "enter" && !w.run {
			return w.install(m)
		}
	case wzReady:
		if ks == "enter" && !w.run {
			if w.err != nil {
				return w.setup(m)
			}
			if w.saved {
				h := m.hostByName(w.host.Name)
				m.modal = nil
				if h != nil {
					m.openHost(h)
					m.modal = newSessionDialog(m, h)
				}
			}
		}
	}
	return nil
}

// back goes one step back, or closes the wizard from the first step.
func (w *wizard) back(m *Model) tea.Cmd {
	w.stop()
	switch {
	case w.step == wzWhere || w.saved:
		m.modal = nil
	case w.step == wzInstall:
		w.step = wzTest
	default:
		w.step, w.err, w.checks = wzWhere, nil, nil
	}
	return nil
}

func (w *wizard) whereKey(m *Model, k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "tab":
		w.field = (w.field + 1) % 3
		return nil
	case "shift+tab":
		w.field = (w.field + 2) % 3
		return nil
	case "up", "down":
		if w.field != 0 {
			step := map[string]int{"up": -1, "down": 1}[k.String()]
			w.field = min(max(w.field+step, 0), 2)
			return nil
		}
		list := w.matches()
		if len(list) == 0 {
			return nil
		}
		if k.String() == "down" {
			w.pick = (w.pick + 1) % len(list)
		} else {
			w.pick = (w.pick - 1 + len(list)) % len(list)
		}
		w.in[0].set(list[w.pick])
		w.follow()
		return nil
	case "enter":
		return w.submit(m)
	}
	if w.in[w.field].key(k) {
		switch w.field {
		case 0:
			w.pick = -1
			w.follow()
		case 1:
			w.named = true
		}
	}
	return nil
}

// submit checks the first step and starts the test.
func (w *wizard) submit(m *Model) tea.Cmd {
	target := strings.TrimSpace(w.in[0].value())
	name := strings.TrimSpace(w.in[1].value())
	w.err = nil
	switch {
	case target == "":
		w.err = &client.UserError{Msg: "an SSH target is needed", Hint: "Type a Host from ~/.ssh/config, or user@host."}
	case !client.ValidHostName(name):
		w.err = &client.UserError{Msg: fmt.Sprintf("%q cannot name a host", name), Hint: "Use letters, digits, dots, dashes and underscores."}
	case m.hostByName(name) != nil:
		w.err = &client.UserError{Msg: "there is already a host named " + name, Hint: "Pick another name, or remove " + name + " first."}
	}
	if w.err != nil {
		return nil
	}
	w.host = client.Host{Name: name, SSH: target, SSHConfig: strings.TrimSpace(w.in[2].value())}
	return w.probe(m)
}

// cliHint is the command that does the same with prompts allowed.
func (w *wizard) cliHint() string {
	s := fmt.Sprintf("%s host add %s", brand.Name, w.host.Name)
	if w.host.SSH != w.host.Name {
		s += " " + w.host.SSH
	}
	if w.host.SSHConfig != "" {
		s += " --ssh-config " + w.host.SSHConfig
	}
	return s + "   (it can ask for passwords)"
}

// sshErr explains a failed SSH step. The wizard cannot show SSH's prompts,
// so the way out is the CLI, which can.
func (w *wizard) sshErr(err error) error {
	msg, hint := errText(err)
	if offline(err) || strings.Contains(msg, "Permission denied") || strings.Contains(msg, "Host key") || hint == "" {
		return &client.UserError{Msg: msg, Hint: w.cliHint()}
	}
	return err
}

func (w *wizard) begin(m *Model) (context.Context, int) {
	w.stop()
	w.run = true
	ctx, cancel := context.WithCancel(m.ctx)
	w.cancel = cancel
	return ctx, w.seq
}

func (w *wizard) probe(m *Model) tea.Cmd {
	w.step, w.err = wzTest, nil
	w.checks = []check{{name: "connect", state: ckRunning}}
	ctx, seq := w.begin(m)
	be, h, clock := m.be, w.host, m.now
	return func() tea.Msg {
		start := clock()
		cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		p, err := be.Probe(cctx, h)
		return probeMsg{seq: seq, p: p, err: err, took: clock().Sub(start)}
	}
}

func (w *wizard) install(m *Model) tea.Cmd {
	w.err = nil
	w.checks = []check{{name: "install", state: ckRunning, detail: "uploading " + brand.HostBinary}}
	ctx, seq := w.begin(m)
	be, h, p, clock := m.be, w.host, w.plat, m.now
	return func() tea.Msg {
		start := clock()
		cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		err := be.Install(cctx, h, p)
		return installMsg{seq: seq, err: err, took: clock().Sub(start)}
	}
}

// setup connects, prepares the host, reads its facts and saves it.
func (w *wizard) setup(m *Model) tea.Cmd {
	w.step, w.err = wzReady, nil
	w.checks = []check{{name: "prepare", state: ckRunning, detail: "linger, terminfo"}}
	ctx, seq := w.begin(m)
	be, h, set, clock := m.be, w.host, m.conns, m.now
	return func() tea.Msg {
		start := clock()
		cctx, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
		c, err := be.Dial(cctx, h)
		if err != nil {
			return setupMsg{seq: seq, err: err}
		}
		if !set.add(c) {
			return nil
		}
		res, err := c.Setup(cctx)
		if err != nil {
			set.drop(c)
			return setupMsg{seq: seq, err: err}
		}
		info, err := c.Doctor(cctx)
		if err != nil {
			set.drop(c)
			return setupMsg{seq: seq, err: err}
		}
		return setupMsg{seq: seq, conn: c, res: res, info: info, took: clock().Sub(start)}
	}
}

// onMsg takes the results of the wizard's commands.
func (w *wizard) onMsg(m *Model, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case probeMsg:
		if msg.seq != w.seq {
			return nil
		}
		w.run, w.cancel = false, nil
		if msg.err != nil {
			w.checks[0].state = ckFailed
			w.err = w.sshErr(msg.err)
			return nil
		}
		w.plat = msg.p
		w.checks = []check{
			{name: "connect", state: ckOK, took: msg.took},
			{name: "system", state: ckOK, detail: msg.p.OS + "/" + msg.p.Arch},
		}
		if _, ok := m.be.HelperSize(msg.p); !ok && !msg.p.Installed {
			w.checks[1].state = ckFailed
			w.err = &client.UserError{
				Msg:  fmt.Sprintf("this build has no helper for %s/%s", msg.p.OS, msg.p.Arch),
				Hint: "Linux hosts on amd64 and arm64 are supported.",
			}
			return nil
		}
		if msg.p.Installed {
			w.checks = append(w.checks, check{name: "helper", state: ckOK, detail: m.be.HelperVersion() + " found"})
		} else {
			w.checks = append(w.checks, check{name: "helper", state: ckNote, detail: "not installed yet"})
		}
		return nil

	case installMsg:
		if msg.seq != w.seq {
			return nil
		}
		w.run, w.cancel = false, nil
		if msg.err != nil {
			w.checks[0].state = ckFailed
			w.err = w.sshErr(msg.err)
			return nil
		}
		w.plat.Installed = true
		return w.setup(m)

	case setupMsg:
		if msg.seq != w.seq {
			if msg.conn != nil {
				return m.dropConn(msg.conn)
			}
			return nil
		}
		if msg.err != nil {
			w.run, w.cancel = false, nil
			w.checks[0].state = ckFailed
			w.err = w.sshErr(msg.err)
			return nil
		}
		w.conn = msg.conn
		info := msg.info
		w.info = &info
		w.checks = []check{{name: "connect", state: ckOK, took: msg.took}}
		for _, s := range msg.res.Steps {
			c := check{name: s.Name, state: ckOK, detail: s.Detail}
			if !s.OK {
				c.state = ckFailed
				if s.Fix != "" {
					c.detail = strings.TrimSpace(c.detail + "  Try: " + s.Fix)
				}
			}
			w.checks = append(w.checks, c)
		}
		return m.saveHosts(w.host, false)

	case hostSavedMsg:
		w.run, w.cancel = false, nil
		if msg.err != nil {
			w.err = msg.err
			return nil
		}
		w.saved = true
		conn := w.conn
		w.conn = nil
		cmd := m.addHost(w.host, conn, w.info)
		if h := m.hostByName(w.host.Name); h != nil {
			m.openHost(h)
		}
		return cmd
	}
	return nil
}

func (w *wizard) view(m *Model) string {
	width := m.dialogWidth()
	cw := width - 4
	title := "Add a host"
	var body []string
	var hints []hint
	switch w.step {
	case wzWhere:
		body, hints = w.whereView(cw)
	case wzTest:
		body = append([]string{"", sBold.Render("Testing " + w.host.Name), ""}, w.checkLines(m, cw)...)
		hints = []hint{{"enter", "next"}, {"esc", "back"}}
		if w.err != nil {
			hints = []hint{{"enter", "try again"}, {"esc", "back"}}
		}
	case wzInstall:
		title = fmt.Sprintf("Install %s on %s?", brand.HostBinary, w.host.Name)
		body = w.installView(m, cw)
		hints = []hint{{"enter", "install"}, {"esc", "back"}}
	case wzReady:
		body = w.readyView(m, cw)
		hints = []hint{{"enter", "start a session"}, {"esc", "close"}}
		if !w.saved {
			hints = []hint{{"esc", "back"}}
			if w.err != nil {
				hints = []hint{{"enter", "try again"}, {"esc", "back"}}
			}
		}
	}
	if w.err != nil {
		body = append(append(body, ""), errorLines(w.err, cw)...)
	}
	body = append(body, "")
	step := sFaint.Render(fmt.Sprintf("step %d of %d", int(w.step)+1, wizSteps))
	return sheet(width, title, step, hints, trimBlank(body), toneNormal)
}

func (w *wizard) whereView(cw int) ([]string, []hint) {
	const lw = 13
	label := func(i int, s string) string {
		if w.field == i {
			return sBold.Render(fit(s, lw))
		}
		return sFaint.Render(fit(s, lw))
	}
	fw := cw - lw
	body := []string{
		"",
		sBold.Render("Where does it live?"),
		"",
		label(0, "SSH target") + w.in[0].view(fw, w.field == 0),
	}
	if list := w.matches(); len(list) > 0 {
		shown := strings.Join(list, ", ")
		count := fmt.Sprintf("  (%d found)", len(list))
		body = append(body, strings.Repeat(" ", lw)+sFaint.Render(trunc("From your SSH config: "+shown, fw-cells(count))+count))
	} else if len(w.known) > 0 {
		body = append(body, strings.Repeat(" ", lw)+sFaint.Render(trunc("Nothing in your SSH config matches.", fw)))
	}
	body = append(body,
		label(1, "Name")+w.in[1].view(fw, w.field == 1),
		label(2, "SSH config")+w.in[2].view(fw, w.field == 2),
	)
	hints := []hint{{"↑↓", "pick from SSH config"}, {"tab", "next field"}, {"enter", "test"}, {"esc", "cancel"}}
	if w.field != 0 {
		hints[0] = hint{"↑↓", "field"}
	}
	return body, hints
}

func (w *wizard) checkLines(m *Model, cw int) []string {
	var out []string
	for _, c := range w.checks {
		var mark string
		switch c.state {
		case ckRunning:
			mark = sKey.Render(m.spinner())
		case ckOK:
			mark = sSuccess.Render(glyphOK)
		case ckFailed:
			mark = sDanger.Render(glyphFail)
		default:
			mark = sFaint.Render(glyphIdle)
		}
		detail := c.detail
		if c.took >= time.Millisecond {
			detail = strings.TrimSpace(detail + " " + fmtLatency(c.took))
		}
		out = append(out, mark+" "+fit(c.name, 12)+sFaint.Render(trunc(detail, cw-14)))
	}
	return out
}

func (w *wizard) installView(m *Model, cw int) []string {
	ver := m.be.HelperVersion()
	size := ""
	if n, ok := m.be.HelperSize(w.plat); ok {
		size = fmt.Sprintf(", about %.0f MB", float64(n)/1e6)
	}
	field := func(k, v string) string { return "  " + sFaint.Render(fit(k, 8)) + trunc(v, cw-10) }
	body := []string{""}
	body = append(body, wrap(brand.DisplayName+" needs a small helper on the host to keep sessions alive.", cw)...)
	body = append(body, "",
		field("What", brand.HostBinary+" "+ver+size+", SHA-256 checked"),
		field("Where", fmt.Sprintf("~/.local/share/%s/bin/%s/", brand.Name, ver)),
		field("Also", "turn on linger, install this terminal's terminfo"),
		"")
	body = append(body, wrap("Nothing listens on the network, and nothing runs until you start a session.", cw)...)
	if w.run || len(w.checks) > 0 && w.checks[0].state == ckFailed {
		body = append(append(body, ""), w.checkLines(m, cw)...)
	}
	return body
}

func (w *wizard) readyView(m *Model, cw int) []string {
	body := append([]string{""}, w.checkLines(m, cw)...)
	if !w.saved || w.info == nil {
		return body
	}
	i := w.info
	var names []string
	for k := range i.Agents {
		names = append(names, k)
	}
	sort.Strings(names)
	field := func(k, v string) string { return "  " + sFaint.Render(fit(k, 8)) + trunc(v, cw-10) }
	sys := []string{}
	for _, s := range []string{i.Distro, i.Arch, plural(i.CPUs, "CPU", "CPUs"), fmtSize(i.MemTotal)} {
		if s != "" && !strings.HasPrefix(s, "0 ") {
			sys = append(sys, s)
		}
	}
	return append(body,
		"",
		sSuccess.Render(glyphOK)+" "+sBold.Render(w.host.Name+" is ready."),
		"",
		field("Agents", strings.Join(names, dot)),
		field("System", strings.Join(sys, dot)),
		field("Limits", fmtSize(i.MemoryMax)+" per session"),
	)
}
