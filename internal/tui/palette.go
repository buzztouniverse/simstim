package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/version"
)

// scope is where a key binding works.
type scope int

const (
	anywhere scope = iota
	inHosts
	inSessions
)

type binding struct {
	key   string
	where scope
}

// command is one entry of the registry. The same registry feeds the
// palette, the help grid and the keys at home (docs/ux.md 2.6).
type command struct {
	noun, verb string
	keys       []binding
	firstRun   bool // works on the first-run screen too
	hidden     bool // kept out of the palette
	why        func(m *Model) string
	run        func(m *Model) tea.Cmd
}

func (c *command) name() string { return c.noun + ": " + c.verb }

// keyLabel is the first key, for the palette and help.
func (c *command) keyLabel() string {
	if len(c.keys) == 0 {
		return ""
	}
	return c.keys[0].key
}

// reason is why the command cannot run now, or "".
func (c *command) reason(m *Model) string {
	if len(m.hosts) == 0 && !c.firstRun {
		return "add a host first"
	}
	if c.why == nil {
		return ""
	}
	return c.why(m)
}

func (c *command) matches(m *Model, key string) bool {
	if len(m.hosts) == 0 && !c.firstRun {
		return false
	}
	for _, b := range c.keys {
		if b.key != key {
			continue
		}
		switch b.where {
		case inHosts:
			if m.focus == paneHosts {
				return true
			}
		case inSessions:
			if m.focus == paneSessions {
				return true
			}
		default:
			return true
		}
	}
	return false
}

func (m *Model) runCommand(c *command) tea.Cmd {
	if r := c.reason(m); r != "" {
		return m.toast(toastInfo, sentence("Not available: "+r))
	}
	return c.run(m)
}

// commands is the registry. It is built in init, because its commands open
// the palette, which reads the registry.
var commands []*command

func init() { commands = registry() }

func registry() []*command {
	needSession := func(m *Model) string {
		if m.selected() == nil {
			return "no session selected"
		}
		return ""
	}
	needHost := func(m *Model) string {
		if m.selectedHost() == nil {
			return "choose a host first"
		}
		return ""
	}
	return []*command{
		{noun: "session", verb: "new", keys: []binding{{"n", anywhere}, {"enter", inHosts}},
			why: func(m *Model) string {
				h := m.newSessionHost()
				switch {
				case h == nil:
					return "choose a host first"
				case h.status == stOffline || h.status == stFailed:
					return h.name() + " is offline"
				}
				return ""
			},
			run: (*Model).openNewSession},
		{noun: "session", verb: "attach", keys: []binding{{"enter", inSessions}},
			why: func(m *Model) string {
				if s := m.selected(); s != nil && !s.Alive {
					return "the session has ended"
				}
				if m.selected() == nil && len(m.rows()) == 0 {
					return ""
				}
				return needSession(m)
			},
			run: (*Model).attachSelected},
		{noun: "session", verb: "rename", keys: []binding{{"r", inSessions}, {"f2", inSessions}}, why: needSession, run: (*Model).startRename},
		{noun: "session", verb: "delete", keys: []binding{{"d", inSessions}}, why: needSession, run: (*Model).confirmDelete},
		{noun: "session", verb: "copy attach command", keys: []binding{{"y", inSessions}}, why: needSession, run: (*Model).copyAttach},

		{noun: "host", verb: "add", keys: []binding{{"a", anywhere}}, firstRun: true, run: (*Model).openWizard},
		{noun: "host", verb: "details", keys: []binding{{"i", inHosts}}, why: needHost,
			run: func(m *Model) tea.Cmd { m.modal = &details{host: m.selectedHost().name()}; return nil }},
		{noun: "host", verb: "test", why: needHost, run: func(m *Model) tea.Cmd { return m.testHost(m.selectedHost()) }},
		{noun: "host", verb: "reconnect", why: needHost, run: func(m *Model) tea.Cmd { return m.reconnect(m.selectedHost()) }},
		{noun: "host", verb: "remove", why: needHost,
			run: func(m *Model) tea.Cmd { m.modal = removeHostDialog(m.selectedHost().host); return nil }},

		{noun: "view", verb: "all hosts", run: func(m *Model) tea.Cmd { m.hostSel = 0; m.syncSel(); return nil }},
		{noun: "view", verb: "filter", keys: []binding{{"/", anywhere}}, run: func(m *Model) tea.Cmd { m.startFilter(); return nil }},
		{noun: "view", verb: "toggle preview", keys: []binding{{"space", anywhere}},
			run: func(m *Model) tea.Cmd {
				m.showPreview = !m.showPreview
				m.syncSel()
				if m.showPreview {
					m.preview.key = sessKey{}
					return m.schedulePreview()
				}
				return nil
			}},
		{noun: "view", verb: "toggle exited", keys: []binding{{"alt+e", anywhere}},
			run: func(m *Model) tea.Cmd { m.hideExited = !m.hideExited; m.syncSel(); return nil }},
		{noun: "view", verb: "shortcuts", keys: []binding{{"?", anywhere}}, firstRun: true,
			run: func(m *Model) tea.Cmd { m.modal = &helpView{}; return nil }},
		{noun: "view", verb: "commands", keys: []binding{{"ctrl+o", anywhere}}, firstRun: true, hidden: true,
			run: func(m *Model) tea.Cmd { m.modal = &palette{in: newInput("", "type a command")}; return nil }},

		{noun: brand.Name, verb: "settings", firstRun: true, run: (*Model).openSettings},
		{noun: brand.Name, verb: "about", firstRun: true,
			run: func(m *Model) tea.Cmd {
				return m.toast(toastInfo, fmt.Sprintf("%s %s, protocol %s, host helper %s.", brand.DisplayName, version.Version, version.Protocol, m.be.HelperVersion()))
			}},
		{noun: brand.Name, verb: "quit", keys: []binding{{"q", anywhere}}, firstRun: true,
			run: func(*Model) tea.Cmd { return tea.Quit }},
	}
}

// Palette.

type palette struct {
	in  input
	sel int
}

type paletteItem struct {
	c      *command
	reason string
}

func (p *palette) items(m *Model) []paletteItem {
	type scored struct {
		paletteItem
		score, order int
	}
	var list []scored
	for i, c := range commands {
		if c.hidden {
			continue
		}
		s, ok := fuzzy(p.in.value(), c.name())
		if !ok {
			continue
		}
		list = append(list, scored{paletteItem{c, c.reason(m)}, s, i})
	}
	if strings.TrimSpace(p.in.value()) != "" {
		sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	}
	out := make([]paletteItem, len(list))
	for i, s := range list {
		out[i] = s.paletteItem
	}
	return out
}

func (p *palette) paste(s string) { p.in.insert(s); p.sel = 0 }

func (p *palette) key(m *Model, k tea.KeyPressMsg) tea.Cmd {
	items := p.items(m)
	switch k.String() {
	case "esc", "ctrl+o":
		m.modal = nil
	case "up", "ctrl+p":
		p.sel = max(p.sel-1, 0)
	case "down", "ctrl+n":
		p.sel = min(p.sel+1, max(len(items)-1, 0))
	case "enter":
		if p.sel >= len(items) {
			return nil
		}
		it := items[p.sel]
		if it.reason != "" {
			return m.toast(toastInfo, sentence("Not available: "+it.reason))
		}
		m.modal = nil
		return it.c.run(m)
	default:
		if p.in.key(k) {
			p.sel = 0
		}
	}
	return nil
}

func (p *palette) view(m *Model) string {
	w := m.dialogWidth(76)
	inner := w - 2
	items := p.items(m)
	listH := min(len(items), max(m.height-8, 3), 12)
	top := max(0, p.sel-listH+1)
	body := []string{" " + sPrimary.Render(">") + " " + p.in.view(inner-4, true)}
	for i := top; i < len(items) && i < top+listH; i++ {
		it := items[i]
		nameSt, right, rightSt := sPlain, it.c.keyLabel(), sKey
		if it.reason != "" {
			nameSt, right, rightSt = sMuted, "not available: "+it.reason, sMuted
		}
		// "  name  right  ": the name takes what the right side leaves.
		right = trunc(right, max(inner-6-16, 0))
		nameW := inner - 6 - cells(right)
		body = append(body, row(i == p.sel, seg{"  ", sPlain}, seg{fit(it.c.name(), nameW), nameSt},
			seg{"  ", sPlain}, seg{right, rightSt}, seg{"  ", sPlain}))
	}
	if len(items) == 0 {
		body = append(body, "  "+sMuted.Render(trunc(fmt.Sprintf("Nothing matches '%s'.", p.in.value()), inner-4)))
	}
	hints := []hint{{"↑↓", "choose"}, {"enter", "run"}, {"esc", "close"}}
	return box(w, len(body)+2, "Commands", "", hintsFor(hints, w), body, boxNormal)
}

func (p *palette) top() bool { return true }

// Help (docs/ux.md 2.7): a grid built from the live registry.

type helpView struct{}

func (helpView) top() bool { return true }

func (h *helpView) key(m *Model, k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "esc", "?", "q", "enter":
		m.modal = nil
	}
	return nil
}

// helpItems are the keys at home, from the registry, after the moves.
func helpItems() [][2]string {
	items := [][2]string{
		{"↑↓ jk", "move"},
		{"tab", "switch pane"},
		{"enter", "new session, or attach"},
		{"esc", "back, clear filter"},
	}
	seen := map[string]bool{"enter": true}
	for _, c := range commands {
		if len(c.keys) == 0 {
			continue
		}
		var keys []string
		for _, b := range c.keys {
			if !seen[b.key] && !slices.Contains(keys, b.key) {
				keys = append(keys, b.key)
			}
		}
		if len(keys) == 0 {
			continue
		}
		items = append(items, [2]string{strings.Join(keys, " "), c.name()})
	}
	return append(items, [2]string{"ctrl+c", "twice to quit"})
}

func (h *helpView) view(m *Model) string {
	w := min(max(m.width-4, 40), 80)
	inner := w - 2
	items := helpItems()
	keyW := 0
	for _, it := range items {
		keyW = max(keyW, cells(it[0]))
	}
	var body []string
	if w < 50 {
		for _, it := range items {
			body = append(body, " "+sKey.Render(it[0]), "   "+sMuted.Render(trunc(it[1], inner-4)))
		}
	} else {
		colW := (inner - 2) / 2
		half := (len(items) + 1) / 2
		cell := func(i int) string {
			if i >= len(items) {
				return ""
			}
			return sKey.Render(fitRight(items[i][0], keyW)) + " " + sMuted.Render(fit(items[i][1], colW-keyW-2))
		}
		for i := range half {
			body = append(body, " "+fit(cell(i), colW)+" "+cell(i+half))
		}
	}
	body = append([]string{""}, append(body, "")...)
	return box(w, min(len(body)+2, max(m.height-2, 6)), "Shortcuts", "", hintsFor([]hint{{"esc", "close"}}, w), body, boxNormal)
}

// Settings open in $EDITOR.

type editorDoneMsg struct{ err error }

func (m *Model) openSettings() tea.Cmd {
	path := filepath.Join(client.ConfigDir(), "config.toml")
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	argv := strings.Fields(editor)
	x := &funcExec{run: func(in, out *os.File) error {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		cmd := exec.Command(argv[0], append(argv[1:], path)...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, os.Stderr
		return cmd.Run()
	}}
	return m.exec(x, func(err error) tea.Msg {
		if err != nil {
			err = &client.UserError{Msg: "the editor failed: " + err.Error(), Hint: "Set $EDITOR, or edit " + path + " by hand."}
		}
		return editorDoneMsg{err: err}
	})
}
