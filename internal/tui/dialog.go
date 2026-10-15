package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/iAmLakshya/simstim/internal/agents"
	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
)

// modal is a notched dialog over the home screen. Only one shows at a time.
type modal interface {
	key(m *Model, k tea.KeyPressMsg) tea.Cmd
	view(m *Model) string
}

// dialogWidth is the width of a dialog on this screen.
func (m *Model) dialogWidth(most int) int { return max(min(m.width-4, most), 30) }

// errorLines draws an error and its "Try:" line inside a dialog.
func errorLines(err error, w int) []string {
	msg, hint := errText(err)
	var out []string
	for i, l := range wrap(sentence(msg), w-4) {
		lead := "  " + sDanger.Render(glyphFail) + " "
		if i > 0 {
			lead = "    "
		}
		out = append(out, lead+l)
	}
	if hint != "" {
		for i, l := range wrap(hint, w-11) {
			lead := "    " + sPrimary.Render("Try:") + " "
			if i > 0 {
				lead = strings.Repeat(" ", 9)
			}
			out = append(out, lead+l)
		}
	}
	return out
}

// New session (docs/ux.md 2.4).

type agentChoice struct {
	kind      string
	installed bool
}

type newSession struct {
	host     string
	choices  []agentChoice
	ai       int
	field    int // 0 agent, 1 folder, 2 name, 3 worktree, 4 command (only for "other")
	command  input
	folder   input
	name     input
	worktree bool
	suggest  []string // folder completions for the current folder text
	hints    bool     // suggest holds recent folders and repositories, not completions
	si       int      // the next suggestion Tab takes
	finding  bool     // the host's agents are not known yet
	starting bool
	err      error
}

const labelW = 11

// agentChoices lists the agents the host has, in the agent table's order,
// then the ones it lacks. The shell is always there.
func agentChoices(info *client.HostInfo) []agentChoice {
	have := map[string]bool{"shell": true}
	if info != nil {
		for k := range info.Agents {
			have[k] = true
		}
	}
	var on, off []agentChoice
	for _, a := range agents.Table {
		switch {
		case have[a.Kind]:
			on = append(on, agentChoice{a.Kind, true})
		case info != nil:
			off = append(off, agentChoice{a.Kind, false})
		}
	}
	// "other" runs any command the user types, for agents not in the table.
	on = append(on, agentChoice{otherAgent, true})
	return append(on, off...)
}

// otherAgent is the choice that runs a command the user types.
const otherAgent = "other…"

func newSessionDialog(m *Model, h *hostConn) *newSession {
	d := &newSession{
		host:    h.name(),
		choices: agentChoices(h.info),
		command: newInput("", "a command, for example aider --model sonnet"),
		folder:  newInput("~", "~"),
		name:    newInput("", "the folder name"),
		finding: h.info == nil,
	}
	// The last choice on this host, or the first agent it has.
	want := m.lastAgent[h.name()]
	for i, c := range d.choices {
		if c.installed && c.kind == want {
			d.ai = i
		}
	}
	return d
}

func (d *newSession) agent() string {
	if d.ai < len(d.choices) {
		return d.choices[d.ai].kind
	}
	return "shell"
}

func (d *newSession) busy() bool { return d.starting }

func (d *newSession) paste(s string) {
	switch d.field {
	case 1:
		d.folder.insert(s)
	case 2:
		d.name.insert(s)
	case 4:
		d.command.insert(s)
	}
}

// order is the sequence Tab moves through; the command field exists only for "other".
func (d *newSession) order() []int {
	if d.agent() == otherAgent {
		return []int{0, 4, 1, 2, 3}
	}
	return []int{0, 1, 2, 3}
}

func (d *newSession) step(n int) {
	o := d.order()
	i := 0
	for j, f := range o {
		if f == d.field {
			i = j
		}
	}
	d.field = o[(i+n+len(o))%len(o)]
}

func (d *newSession) top() bool { return false }

// refresh picks up the host's agents once the doctor has answered.
func (d *newSession) refresh(m *Model) {
	h := m.hostByName(d.host)
	if !d.finding || h == nil || h.info == nil {
		return
	}
	cur := d.agent()
	d.choices, d.finding = agentChoices(h.info), false
	d.ai = 0
	want := m.lastAgent[d.host]
	for i, c := range d.choices {
		if c.installed && (c.kind == cur && want == "" || c.kind == want) {
			d.ai = i
		}
	}
}

func (d *newSession) moveAgent(step int) {
	for i := d.ai + step; i >= 0 && i < len(d.choices); i += step {
		if d.choices[i].installed {
			d.ai = i
			return
		}
	}
}

func (d *newSession) key(m *Model, k tea.KeyPressMsg) tea.Cmd {
	d.refresh(m)
	ks := k.String()
	if d.starting {
		if ks == "esc" {
			m.modal = nil
		}
		return nil
	}
	if d.field == 1 && ks == "tab" {
		// Tab completes a typed path from the host's matches, cycling through
		// them; with nothing left to complete it moves to the next field.
		if next, ok := d.completion(); ok {
			d.folder.set(next)
			return m.fetchFolders(d)
		}
	}
	switch ks {
	case "esc":
		m.modal = nil
		return nil
	case "tab", "down":
		d.step(1)
		if d.field == 1 {
			return m.fetchFolders(d)
		}
		return nil
	case "shift+tab", "up":
		d.step(-1)
		return nil
	case "enter", "ctrl+enter", "alt+enter", "ctrl+j":
		h := m.hostByName(d.host)
		if h == nil {
			m.modal = nil
			return nil
		}
		folder := strings.TrimSpace(d.folder.value())
		if folder == "" {
			folder = "~"
		}
		a := client.NewArgs{Agent: d.agent(), Cwd: client.RemotePath(folder), Name: strings.TrimSpace(d.name.value()), Worktree: d.worktree}
		if d.agent() == otherAgent {
			c := strings.TrimSpace(d.command.value())
			if c == "" {
				d.field, d.err = 4, &client.UserError{Msg: "type the command to run", Hint: "For example: aider --model sonnet"}
				return nil
			}
			// exec replaces the shell, so the command is the session's process.
			a.Agent, a.Argv = "", []string{"/bin/sh", "-c", "exec " + c}
		}
		d.starting, d.err = true, nil
		return m.startSession(h, a, ks == "enter")
	}
	switch d.field {
	case 0:
		switch ks {
		case "left", "h":
			d.moveAgent(-1)
		case "right", "l", "space":
			d.moveAgent(1)
		default:
			// A letter jumps to the next agent that starts with it.
			if len(k.Text) == 1 {
				for i := 1; i <= len(d.choices); i++ {
					j := (d.ai + i) % len(d.choices)
					if c := d.choices[j]; c.installed && strings.HasPrefix(c.kind, k.Text) {
						d.ai = j
						break
					}
				}
			}
		}
	case 1:
		if d.folder.key(k) {
			return m.fetchFolders(d)
		}
	case 2:
		d.name.key(k)
	case 4:
		d.command.key(k)
	case 3:
		switch ks {
		case "left", "h":
			d.worktree = false
		case "right", "l":
			d.worktree = true
		case "space":
			d.worktree = !d.worktree
		}
	}
	return nil
}

func (d *newSession) view(m *Model) string {
	w := m.dialogWidth(76)
	inner := w - 2
	label := func(i int, s string) string {
		if d.field == i {
			return "  " + sBold.Render(fit(s, labelW))
		}
		return "  " + sMuted.Render(fit(s, labelW))
	}
	agentsLine, under := d.agentRow(inner - 4 - labelW)
	body := []string{
		"",
		label(0, "Agent") + agentsLine,
		d.secondRow(label, under, inner-4-labelW),
		label(1, "Folder") + d.folder.view(inner-4-labelW, d.field == 1),
		"  " + strings.Repeat(" ", labelW) + d.suggestLine(inner-4-labelW),
		label(2, "Name") + d.name.view(inner-4-labelW, d.field == 2),
		label(3, "Worktree") + d.worktreeRow(),
		"",
	}
	switch {
	case d.starting:
		body = append(body, "  "+sPrimary.Render(m.spinner())+" "+sMuted.Render(fmt.Sprintf("Starting %s on %s…", d.agent(), d.host)), "")
	case d.err != nil:
		body = append(append(body, errorLines(d.err, inner)...), "")
	}
	hints := []hint{{"enter", "start"}, {"tab", "next field"}, {"ctrl+enter", "start detached"}, {"esc", "cancel"}}
	return box(w, len(body)+2, "New session on "+d.host, "", hintsFor(hints, w), body, boxNormal)
}

// worktreeRow is a choice of two: the folder itself, or a new git worktree.
func (d *newSession) worktreeRow() string {
	opt := func(on bool, s string) string {
		mark, st := "○", sMuted
		if on {
			mark, st = "●", sBold
			if d.field == 3 {
				st = sKey
			}
		}
		return st.Render(mark + " " + s)
	}
	return opt(!d.worktree, "this folder") + "   " + opt(d.worktree, "new worktree")
}

// agentRow draws the agent choices and the mark under the chosen one.
func (d *newSession) agentRow(w int) (string, string) {
	const gap = 3
	var b, u strings.Builder
	used := 0
	var missing []string
	for i, c := range d.choices {
		if !c.installed {
			missing = append(missing, c.kind)
			continue
		}
		n := cells(c.kind)
		if used+n > w {
			b.WriteString(sMuted.Render("…"))
			break
		}
		if used > 0 {
			b.WriteString(strings.Repeat(" ", gap))
			u.WriteString(strings.Repeat(" ", gap))
			used += gap
		}
		switch {
		case i == d.ai && d.field == 0:
			b.WriteString(sKey.Render(c.kind))
			u.WriteString(sPrimary.Render(strings.Repeat("▔", n)))
		case i == d.ai:
			b.WriteString(sBold.Render(c.kind))
			u.WriteString(sMuted.Render(strings.Repeat("▔", n)))
		default:
			b.WriteString(c.kind)
			u.WriteString(strings.Repeat(" ", n))
		}
		used += n
	}
	if d.finding {
		if room := w - used - gap; room > 8 {
			b.WriteString(strings.Repeat(" ", gap) + sMuted.Render(trunc("finding agents…", room)))
		}
	} else if len(missing) > 0 {
		if room := w - used - gap; room > 16 {
			b.WriteString(strings.Repeat(" ", gap) + sMuted.Render(trunc("not installed: "+strings.Join(missing, ", "), room)))
		}
	}
	return b.String(), u.String()
}

// Confirm (docs/ux.md 2.5).

type confirm struct {
	title  string
	lines  []string
	yes    hint
	no     hint
	danger bool
	run    func(m *Model) tea.Cmd
}

func (c *confirm) key(m *Model, k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "enter", "y":
		m.modal = nil
		return c.run(m)
	case "esc", "n", "q":
		m.modal = nil
	}
	return nil
}

func (c *confirm) view(m *Model) string {
	w := m.dialogWidth(76)
	body := []string{""}
	for _, l := range c.lines {
		for _, x := range wrap(l, w-6) {
			body = append(body, "  "+x)
		}
	}
	body = append(body, "")
	bs := boxNormal
	if c.danger {
		bs = boxDanger
	}
	return box(w, len(body)+2, c.title, "", hintsFor([]hint{c.yes, c.no}, w), body, bs)
}

func deleteDialog(h *hostConn, s client.Session) *confirm {
	host := h.name()
	lines := []string{
		fmt.Sprintf("This stops %s on %s and removes the session.", s.Agent, host),
		fmt.Sprintf("The folder %s stays. Unsaved work in the agent is lost.", homePath(h, s.Cwd)),
	}
	if !s.Alive {
		lines = []string{
			fmt.Sprintf("This removes the session from %s.", host),
			fmt.Sprintf("The folder %s stays.", homePath(h, s.Cwd)),
		}
	}
	return &confirm{
		title: "Delete " + s.Name + "?", lines: lines, danger: true,
		yes: hint{"enter", "delete"}, no: hint{"esc", "keep"},
		run: func(m *Model) tea.Cmd { return m.deleteSession(host, s) },
	}
}

func removeHostDialog(h client.Host) *confirm {
	return &confirm{
		title: "Remove " + h.Name + "?",
		lines: []string{
			fmt.Sprintf("This removes %s from the hosts on this computer.", h.Name),
			"Its sessions and the helper stay on the host, and keep running.",
		},
		danger: true, yes: hint{"enter", "remove"}, no: hint{"esc", "keep"},
		run: func(m *Model) tea.Cmd { return m.saveHosts(h, true) },
	}
}

// Host details (docs/ux.md 2.9).

type details struct{ host string }

func (d *details) key(m *Model, k tea.KeyPressMsg) tea.Cmd {
	h := m.hostByName(d.host)
	if h == nil {
		m.modal = nil
		return nil
	}
	switch k.String() {
	case "esc", "q":
		m.modal = nil
	case "t":
		return m.testHost(h)
	case "r":
		return m.reconnect(h)
	case "x":
		m.modal = removeHostDialog(h.host)
	case "s":
		return m.openShell(h)
	}
	return nil
}

func (d *details) view(m *Model) string {
	w := m.dialogWidth(76)
	h := m.hostByName(d.host)
	if h == nil {
		return ""
	}
	field := func(k, v string) string { return "  " + sMuted.Render(fit(k, labelW)) + v }
	route := h.host.SSH
	if h.host.SSHConfig != "" {
		route += sMuted.Render("  with " + h.host.SSHConfig)
	}
	body := []string{field("Route", route)}
	if i := h.info; i != nil {
		sys := []string{}
		for _, s := range []string{i.Distro, i.Arch, plural(i.CPUs, "CPU", "CPUs"), fmtSize(i.MemTotal), fmtSize(i.DiskFree) + " free"} {
			if s != "" && !strings.HasPrefix(s, "0 ") {
				sys = append(sys, s)
			}
		}
		var names []string
		for k := range i.Agents {
			names = append(names, k)
		}
		sort.Strings(names)
		body = append(body,
			field("System", strings.Join(sys, dot)),
			field("Helper", brand.HostBinary+" "+i.Version),
			field("Agents", strings.Join(names, dot)),
			field("Limits", fmtSize(i.MemoryMax)+" per session"+dot+plural(len(h.sessions), "session", "sessions")))
	} else {
		msg, _ := errText(h.err)
		if msg == "" {
			msg = "waiting for the host to answer"
		}
		body = append(body, field("Status", sMuted.Render(sentence(msg))))
	}
	state := "online" + dot + fmtLatency(h.latency)
	switch h.status {
	case stConnecting:
		state = "connecting"
	case stOffline:
		state = "offline"
	case stFailed:
		state = "not ready"
	}
	hints := []hint{{"t", "test"}, {"s", "shell"}, {"r", "reconnect"}, {"x", "remove"}, {"esc", "back"}}
	return box(w, len(body)+2, h.name(), sMuted.Render(strings.TrimSuffix(state, dot)), hintsFor(hints, w), body, boxNormal)
}

// fetchFolders asks the host for completions of the folder field.
func (m *Model) fetchFolders(d *newSession) tea.Cmd {
	h := m.hostByName(d.host)
	if h == nil || !h.online() {
		return nil
	}
	prefix, name, c, ctx := d.folder.value(), h.name(), h.conn, m.ctx
	return func() tea.Msg {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		res, err := c.Folders(cctx, prefix)
		if err != nil {
			return nil
		}
		return foldersMsg{host: name, prefix: prefix, res: res}
	}
}

// setSuggestions keeps completions that still fit what the user has typed.
func (d *newSession) setSuggestions(prefix string, res client.FoldersResult) {
	if prefix != d.folder.value() {
		return // the user typed on; a newer answer is coming
	}
	var out []string
	seen := map[string]bool{}
	add := func(list []string) {
		for _, s := range list {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	d.hints = prefix == "" || prefix == "~"
	if d.hints {
		add(res.Recent)
		add(res.Repos)
	} else {
		add(res.Matches)
	}
	d.suggest, d.si = out, 0
}

// suggestLine shows the completions under the folder field while it has focus.
func (d *newSession) suggestLine(w int) string {
	if d.field != 1 || len(d.suggest) == 0 {
		return ""
	}
	lead := "↹ "
	if d.hints {
		lead = "recent: "
	}
	return sMuted.Render(fit(lead+strings.Join(d.suggest, "  "), w))
}

// completion returns the next match that extends what the user typed. Adding
// only a trailing slash does not count, and the hints shown for an empty
// field (recent folders, repositories) are not taken by Tab.
func (d *newSession) completion() (string, bool) {
	v := d.folder.value()
	if v == "" || v == "~" || d.hints {
		return "", false
	}
	for range d.suggest {
		next := d.suggest[d.si%len(d.suggest)]
		d.si++
		if next != v && next != v+"/" && strings.HasPrefix(next, strings.TrimSuffix(v, "/")) {
			return next, true
		}
	}
	return "", false
}

// secondRow is the mark under the chosen agent, or, for "other", the command field.
func (d *newSession) secondRow(label func(int, string) string, under string, w int) string {
	if d.agent() == otherAgent {
		return label(4, "Command") + d.command.view(w, d.field == 4)
	}
	return "  " + strings.Repeat(" ", labelW) + under
}

// shellDoneMsg reports that an SSH shell on a host ended.
type shellDoneMsg struct {
	host string
	err  error
}

// openShell hands the terminal to an interactive SSH shell on the host.
func (m *Model) openShell(h *hostConn) tea.Cmd {
	s := &client.SSH{Host: h.host, Interactive: true}
	cmd, err := s.Shell(m.ctx)
	if err != nil {
		return m.toastErr(err)
	}
	name := h.name()
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return shellDoneMsg{host: name, err: err} })
}
