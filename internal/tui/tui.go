// Package tui is the launcher: hosts, then sessions, one list at a time, with
// dialogs and the command palette over it (docs/ux.md).
//
// The model never blocks: every request to a host runs in a tea.Cmd and
// reports back with a message. Attaching hands the whole terminal to
// client.Attach through tea.Exec, and the launcher comes back when the
// user detaches.
package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"golang.org/x/term"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
)

// Run opens the launcher.
func Run(ctx context.Context) error { return RunWith(ctx, Options{}) }

// RunWith opens the launcher with options; Motion is set from the settings.
func RunWith(ctx context.Context, o Options) error { return runWith(ctx, realBackend{}, o) }

func runWith(ctx context.Context, be Backend, o Options) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return &client.UserError{Msg: "the launcher needs a terminal", Hint: brand.Name + " ls, or run it in a terminal"}
	}
	o.Motion = motionOn(be.Settings().UI.Motion, os.Getenv)
	m := New(ctx, be, o)
	defer m.Close()
	_, err := tea.NewProgram(m, tea.WithContext(ctx)).Run()
	if err != nil && ctx.Err() != nil {
		return nil // stopped by a signal; sessions keep running
	}
	return err
}

// motionOn applies ui.motion. "auto" turns motion off inside SSH sessions,
// in Emacs and JetBrains terminals, and with NO_COLOR or TERM=dumb.
func motionOn(setting string, getenv func(string) string) bool {
	switch setting {
	case "on":
		return true
	case "off":
		return false
	}
	switch {
	case getenv("SSH_CONNECTION") != "", getenv("SSH_TTY") != "",
		getenv("INSIDE_EMACS") != "",
		strings.Contains(getenv("TERMINAL_EMULATOR"), "JetBrains"),
		getenv("NO_COLOR") != "", getenv("TERM") == "dumb":
		return false
	}
	return true
}

// View draws the screen. The tab title counts the sessions that need you,
// so another tab can see it.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.ReportFocus = true
	v.WindowTitle = brand.DisplayName
	if h := m.selectedHost(); h != nil && m.screen == scrSessions {
		v.WindowTitle += " · " + h.name()
	}
	if n := m.needYou(); n > 0 {
		v.WindowTitle = fmt.Sprintf("● %d · %s", n, v.WindowTitle)
	}
	return v
}

// needYou counts the sessions that wait for the user, on every host.
func (m *Model) needYou() int {
	n := 0
	for _, h := range m.hosts {
		for _, s := range h.sessions {
			if s.State == "needs_input" {
				n++
			}
		}
	}
	return n
}

// render draws the whole screen: the launcher or the first-run screen with
// their motion, a dialog over a dimmed backdrop, the notices and the quit
// popup.
func (m *Model) render() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	c := lipgloss.NewCanvas(m.width, m.height)
	if len(m.hosts) == 0 {
		uv.NewStyledString(m.viewFirstRun()).Draw(c, c.Bounds())
	} else {
		uv.NewStyledString(m.viewLauncher()).Draw(c, c.Bounds())
		g := m.geo
		for i := range g.listN { // rows fade in, one after another, then the detail line
			m.fade(c, g.x+2, g.listY+i, g.w-4, 1, m.rowAlpha(i))
		}
		if g.detailY > 0 {
			m.fade(c, g.x+2, g.detailY, g.w-4, 1, m.rowAlpha(g.listN+1))
		}
		if gl, ok := m.glowAt(g.x, g.y, g.w, g.h, m.modal == nil && m.glowing(), time.Time{}); ok {
			m.applyGlow(c, gl)
		}
	}
	if m.modal != nil {
		dim(c)
	}
	if len(m.hosts) > 0 {
		for i, n := range m.noticeLines(m.cw()) {
			x, y := m.geo.x+2, m.geo.y+m.geo.h+1+i
			if y >= m.height {
				break
			}
			uv.NewStyledString(n.s).Draw(c, uv.Rect(x, y, m.width-x, 1))
			m.fade(c, x, y, cells(n.s), 1, n.alpha)
		}
	}
	if m.modal != nil {
		// A dialog sits in the launcher's column, from the box's top row.
		b := m.modal.view(m)
		bw, bh := lipgloss.Size(b)
		x, y := max((m.width-bw)/2, 0), m.boxTop()
		if len(m.hosts) == 0 || y+bh > m.height {
			y = max((m.height-bh)/2, 0)
		}
		uv.NewStyledString(b).Draw(c, uv.Rect(x, y, bw, bh))
		busy := false
		if b, ok := m.modal.(interface{ busy() bool }); ok {
			busy = b.busy()
		}
		if gl, ok := m.glowAt(x, y, bw, bh, busy, m.modalAt); ok {
			m.applyGlow(c, gl)
		}
		m.modalGeo = boxGeo{x: x, y: y, w: bw, h: bh}
	}
	if q, ok := m.quitLayer(); ok {
		qw, qh := lipgloss.Size(q.s)
		uv.NewStyledString(q.s).Draw(c, uv.Rect(q.x, q.y, qw, qh))
		if gl, ok := m.glowAt(q.x, q.y, qw, qh, false, m.quitAt); ok {
			m.applyGlow(c, gl)
		}
	}
	return c.Render()
}

// Attach.

// funcExec is a tea.ExecCommand that runs a function with the terminal.
type funcExec struct {
	in, out *os.File
	run     func(in, out *os.File) error
}

func (f *funcExec) SetStdin(r io.Reader) {
	if file, ok := r.(*os.File); ok {
		f.in = file
	}
}

func (f *funcExec) SetStdout(w io.Writer) {
	if file, ok := w.(*os.File); ok {
		f.out = file
	}
}

func (f *funcExec) SetStderr(io.Writer) {}

func (f *funcExec) Run() error {
	in, out := f.in, f.out
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return f.run(in, out)
}

// attachTo hands the terminal to the session until the user detaches.
func (m *Model) attachTo(h *hostConn, id, name string) tea.Cmd {
	o := client.AttachOptions{
		Host: h.host, Session: id, Settings: m.settings, State: m.attach,
		// The launcher has let go of the terminal, so SSH may prompt.
		Interactive: true,
	}
	m.mem.Last = &lastSession{Host: h.name(), ID: id, Name: name, At: m.now()}
	save := m.remember()
	var res client.AttachResult
	var aerr error
	be, ctx, host := m.be, m.ctx, h.name()
	x := &funcExec{run: func(in, out *os.File) error {
		o.In, o.Out = in, out
		res, aerr = be.Attach(ctx, o)
		return nil
	}}
	return tea.Batch(func() tea.Msg { save(); return nil }, m.exec(x, func(err error) tea.Msg {
		if aerr != nil {
			err = aerr
		}
		return attachDoneMsg{host: host, id: id, name: name, res: res, err: err}
	}))
}

func (m *Model) onAttachDone(msg attachDoneMsg) tea.Cmd {
	var cmds []tea.Cmd
	if h := m.hostByName(msg.host); h != nil {
		cmds = append(cmds, m.listSessions(h))
	}
	if msg.err != nil {
		return tea.Batch(append(cmds, m.toastErr(msg.err))...)
	}
	r := msg.res
	if r.Reconnects > 0 {
		text := fmt.Sprintf("Reconnected to %s.", msg.host)
		if r.Missed > 0 {
			text = fmt.Sprintf("Reconnected to %s (missed %s, replayed).", msg.host, fmtSize(r.Missed))
		}
		cmds = append(cmds, m.toast(toastInfo, text))
	}
	var text string
	kind := toastSuccess
	switch e := r.Ended; {
	case e == nil:
		text = fmt.Sprintf("Detached. %s keeps running on %s.", msg.name, msg.host)
	case e.ExitCode == -1:
		text, kind = fmt.Sprintf("%s on %s stopped.", msg.name, msg.host), toastWarning
	case e.OOM:
		text, kind = fmt.Sprintf("%s on %s ended (out of memory).", msg.name, msg.host), toastWarning
	case e.Signal != "":
		text, kind = fmt.Sprintf("%s on %s ended (%s).", msg.name, msg.host, e.Signal), toastWarning
	default:
		text = fmt.Sprintf("%s on %s ended (exit %d).", msg.name, msg.host, e.ExitCode)
		if e.ExitCode != 0 {
			kind = toastWarning
		}
	}
	return tea.Batch(append(cmds, m.toast(kind, text))...)
}
