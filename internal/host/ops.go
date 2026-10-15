package host

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/iAmLakshya/simstim/internal/agents"
	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/paths"
	"github.com/iAmLakshya/simstim/internal/proto"
	"github.com/iAmLakshya/simstim/internal/version"
)

// UserError is an error with a code and a suggested next step.
func UserError(code, msg, hint string) *proto.Error {
	return &proto.Error{Code: code, Message: msg, Hint: hint}
}

type NewArgs = api.NewArgs

type SessionInfo = api.SessionInfo

const idAlphabet = "abcdefghijklmnopqrstuvwxyz234567"

// NewID makes a short random session ID.
func NewID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	out := make([]byte, 8)
	for i := range b {
		out[i] = idAlphabet[int(b[i])%len(idAlphabet)]
	}
	return string(out)
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func expandHome(p string) string {
	if p == "" || p == "~" {
		h, _ := os.UserHomeDir()
		return h
	}
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, rest)
	}
	return p
}

// New starts a session and returns its row.
func New(a NewArgs) (*SessionInfo, error) {
	var argv []string
	kind := a.Agent
	agent, known := agents.Find(a.Agent)
	switch {
	case len(a.Argv) > 0:
		argv = a.Argv
		if kind == "" {
			kind = "custom"
		}
	case known:
		argv = agent.Launch
	case a.Agent == "":
		agent, _ = agents.Find("shell")
		argv, kind = agent.Launch, "shell"
	default:
		return nil, UserError("unknown_agent", fmt.Sprintf("unknown agent %q", a.Agent), "Pick one of the listed agents, or pass a command.")
	}
	cwd := expandHome(a.Cwd)
	if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
		return nil, UserError("bad_folder", fmt.Sprintf("the folder %s does not exist on this host", cwd), "Pick an existing folder.")
	}
	login, err := LoginEnv(15 * time.Second)
	if err != nil {
		login = os.Environ()
	}
	id := NewID()
	term := a.Term
	if !HasTerminfo(term) {
		term = "xterm-256color"
	}
	colorterm := a.ColorTerm
	if colorterm == "" {
		colorterm = "truecolor"
	}
	env := AgentEnv(login, map[string]string{
		"TERM": term, "COLORTERM": colorterm,
		"TERM_PROGRAM": a.TermProgram, "TERM_PROGRAM_VERSION": a.TermProgramVersion,
		brand.Env("SESSION"): id, brand.Env("SOCK"): paths.Socket(id),
	})
	vars := map[string]string{"session_uuid": newUUID()}
	argv = agents.Expand(argv, vars)
	for i, s := range argv {
		if s == "$SHELL" {
			argv[i] = LoginShell()
		}
	}
	if strings.HasPrefix(argv[0], "~/") {
		argv[0] = expandHome(argv[0])
	}
	var hooks []string
	if known && len(agent.Hooks) > 0 && len(a.Argv) == 0 {
		hooks = hookArgs(agent.Hooks)
		argv = append(argv, hooks...)
	}
	if _, err := lookPath(argv[0], envValue(env, "PATH")); err != nil {
		return nil, UserError("agent_missing", fmt.Sprintf("%s is not installed on this host", argv[0]), "Install it on the host, then try again.")
	}
	name := strings.TrimSpace(a.Name)
	if name == "" {
		name = filepath.Base(cwd)
	}
	memMax := a.MemoryMax
	if memMax <= 0 {
		memMax = DefaultMemoryMax()
	}
	now := time.Now().UTC()
	m := &Manifest{
		Schema: 1, ID: id, Name: name, CreatedAt: now, HostVersion: version.Version,
		Agent: AgentInfo{Kind: kind, Argv: argv},
		Cwd:   cwd, Env: env,
		Term:    TermInfo{Term: term, Cols: a.Cols, Rows: a.Rows},
		Limits:  Limits{MemoryMax: memMax},
		Run:     Runtime{Socket: paths.Socket(id), BootID: BootID()},
		Status:  Status{State: StateStarting, Since: now},
		Restart: Restart{Policy: "resume", MaxAttempts: 3},
	}
	if known {
		m.Agent.ResumeArgv = agent.Resume
		m.Agent.ContinueArgv = agent.Continue
		m.Agent.Hooks = hooks
		if agent.IDCapture == "choose" {
			m.Agent.AgentSessionID = vars["session_uuid"]
			m.Agent.IDSource = "chosen"
		}
	}
	if err := paths.EnsurePrivateDir(paths.SessionDir(id)); err != nil {
		return nil, err
	}
	if err := m.Save(); err != nil {
		return nil, err
	}
	if err := launchHolder(m); err != nil {
		return nil, UserError("start_failed", err.Error(), "Check the session log with the host doctor.")
	}
	if mm, err := LoadManifest(id); err == nil {
		mm.Run.Unit = m.Run.Unit
		_ = mm.Save()
		m = mm
	}
	info := infoFrom(m, true)
	return &info, nil
}

func infoFrom(m *Manifest, alive bool) SessionInfo {
	st := m.Status
	if !alive && st.State != StateExited && st.State != StateErrored {
		st.State = StateLost
		if m.Run.BootID != "" && m.Run.BootID != BootID() {
			st.Reason = "the host restarted"
		} else {
			st.Reason = "the session's holder stopped"
		}
	}
	return SessionInfo{
		ID: m.ID, Name: m.Name, Agent: m.Agent.Kind, Cwd: m.Cwd,
		State: st.State, Since: st.Since, Reason: st.Reason, ExitCode: st.ExitCode, Signal: st.Signal, OOM: st.OOM,
		CreatedAt: m.CreatedAt, LastActivityAt: m.LastActivityAt, LastAttachedAt: m.LastAttachedAt,
		Alive: alive, Unit: m.Run.Unit, MemoryMax: m.Limits.MemoryMax, MemoryCurrent: memoryCurrent(m.Run.Unit),
		HostVersion: m.HostVersion, Cols: m.Term.Cols, Rows: m.Term.Rows,
	}
}

func memoryCurrent(unit string) int64 {
	if unit == "" {
		return 0
	}
	uid := os.Getuid()
	p := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/%s/%s/memory.current", uid, uid, brand.Slice, unit)
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	var n int64
	fmt.Sscan(strings.TrimSpace(string(b)), &n)
	return n
}

func alive(id string) bool {
	c, err := net.DialTimeout("unix", paths.Socket(id), 300*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// List returns every session on this host, newest first.
func List() ([]SessionInfo, error) {
	entries, err := os.ReadDir(paths.SessionsDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SessionInfo{}, nil
		}
		return nil, err
	}
	out := make([]SessionInfo, 0, len(entries))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := LoadManifest(id)
			if err != nil {
				return
			}
			info := infoFrom(m, alive(id))
			mu.Lock()
			out = append(out, info)
			mu.Unlock()
		}()
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// HolderRequest sends one request to a session's holder.
func HolderRequest(id, op string, args any, timeout time.Duration) (json.RawMessage, error) {
	conn, err := net.DialTimeout("unix", paths.Socket(id), timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	w := proto.NewWriter(conn)
	r := bufio.NewReader(conn)
	if err := w.WriteJSON(proto.THello, proto.Hello{Magic: proto.Magic, Protocol: version.Protocol, Version: version.Version, Role: "bridge"}); err != nil {
		return nil, err
	}
	if f, err := proto.Read(r); err != nil || f.Type != proto.THello {
		return nil, errors.New("holder did not answer")
	}
	req := proto.Request{ID: 1, Op: op}
	if args != nil {
		req.Args, _ = json.Marshal(args)
	}
	if err := w.WriteJSON(proto.TRequest, req); err != nil {
		return nil, err
	}
	for {
		f, err := proto.Read(r)
		if err != nil {
			return nil, err
		}
		if f.Type != proto.TResponse {
			continue
		}
		var resp proto.Response
		if err := json.Unmarshal(f.Payload, &resp); err != nil {
			return nil, err
		}
		if !resp.OK {
			if resp.Error != nil {
				return nil, resp.Error
			}
			return nil, errors.New("request failed")
		}
		return resp.Result, nil
	}
}

// Delete stops a session and removes its files.
func Delete(id string) error {
	if !validID(id) {
		return UserError("not_found", "no session "+id, "List sessions to find its ID.")
	}
	if _, err := os.Stat(paths.SessionDir(id)); err != nil {
		return UserError("not_found", "no session "+id, "List sessions to find its ID.")
	}
	m, _ := LoadManifest(id)
	if alive(id) {
		_, _ = HolderRequest(id, "kill", nil, 2*time.Second)
		deadline := time.Now().Add(5 * time.Second)
		for alive(id) && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if m != nil && m.Run.Unit != "" {
		_ = exec.Command("systemctl", "--user", "stop", m.Run.Unit).Run()
	} else if m != nil && m.Run.HolderPID > 0 && alive(id) {
		_ = syscall.Kill(m.Run.HolderPID, syscall.SIGTERM)
	}
	os.Remove(paths.Socket(id))
	return os.RemoveAll(paths.SessionDir(id))
}

// Rename changes a session's name.
func Rename(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return UserError("bad_name", "a name is needed", "")
	}
	if alive(id) {
		_, err := HolderRequest(id, "rename", map[string]string{"name": name}, 2*time.Second)
		return err
	}
	m, err := LoadManifest(id)
	if err != nil {
		return UserError("not_found", "no session "+id, "List sessions to find its ID.")
	}
	m.Name = name
	return m.Save()
}

func validID(id string) bool {
	if len(id) == 0 || len(id) > 32 {
		return false
	}
	for _, c := range id {
		if !strings.ContainsRune(idAlphabet, c) {
			return false
		}
	}
	return true
}

type Check = api.Check

type HostInfo = api.HostInfo

// Doctor inspects the host.
func Doctor() HostInfo {
	hn, _ := os.Hostname()
	u, _ := user.Current()
	info := HostInfo{Hostname: hn, OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU(),
		MemTotal: MemTotal(), Version: version.Version, Agents: map[string]string{}, MemoryMax: DefaultMemoryMax()}
	if u != nil {
		info.User = u.Username
	}
	if b, err := exec.Command("uname", "-r").Output(); err == nil {
		info.Kernel = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(l, "PRETTY_NAME="); ok {
				info.Distro = strings.Trim(v, `"`)
			}
		}
	}
	var st syscall.Statfs_t
	if h, err := os.UserHomeDir(); err == nil && syscall.Statfs(h, &st) == nil {
		info.DiskFree = int64(st.Bavail) * int64(st.Bsize)
	}
	login, err := LoginEnv(15 * time.Second)
	if err != nil {
		login = os.Environ()
	}
	path := envValue(login, "PATH")
	for _, a := range agents.Table {
		for _, b := range a.Binaries {
			if b == "$SHELL" {
				info.Agents[a.Kind] = LoginShell()
				break
			}
			if p, err := lookPath(b, path); err == nil {
				info.Agents[a.Kind] = p
				break
			}
		}
	}
	sd := SystemdUser()
	info.Checks = append(info.Checks, Check{Name: "systemd user manager", OK: sd,
		Detail: map[bool]string{true: "sessions run as user services with memory caps", false: "sessions run as detached processes without memory caps"}[sd]})
	if sd {
		linger := lingerOn()
		info.Checks = append(info.Checks, Check{Name: "linger", OK: linger,
			Detail: map[bool]string{true: "sessions survive logout", false: "sessions stop when your last login ends"}[linger],
			Fix:    "loginctl enable-linger $USER"})
		mem := memoryDelegated()
		info.Checks = append(info.Checks, Check{Name: "memory caps", OK: mem,
			Detail: map[bool]string{true: "cgroup v2 memory controller is delegated", false: "the memory controller is not delegated to the user manager"}[mem]})
	}
	gt := HasTerminfo("xterm-ghostty")
	info.Checks = append(info.Checks, Check{Name: "Ghostty terminfo", OK: gt,
		Detail: map[bool]string{true: "xterm-ghostty is installed", false: "xterm-ghostty is missing; sessions fall back to xterm-256color"}[gt],
		Fix:    "infocmp -x xterm-ghostty | ssh <host> tic -x -"})
	info.Checks = append(info.Checks, Check{Name: "runtime folder", OK: true, Detail: paths.RuntimeDir()})
	return info
}

func lingerOn() bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	out, err := exec.Command("loginctl", "show-user", u.Username, "-p", "Linger", "--value").Output()
	return err == nil && strings.TrimSpace(string(out)) == "yes"
}

func memoryDelegated() bool {
	uid := os.Getuid()
	b, err := os.ReadFile(fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/cgroup.controllers", uid, uid))
	return err == nil && strings.Contains(string(b), "memory")
}

type SetupArgs = api.SetupArgs

type SetupResult = api.SetupResult

// Setup turns on linger and installs terminfo entries.
func Setup(a SetupArgs) SetupResult {
	var res SetupResult
	if a.Linger && SystemdUser() {
		if lingerOn() {
			res.Steps = append(res.Steps, Check{Name: "linger", OK: true, Detail: "already on"})
		} else {
			u, _ := user.Current()
			err := exec.Command("loginctl", "enable-linger", u.Username).Run()
			if err != nil {
				err = exec.Command("sudo", "-n", "loginctl", "enable-linger", u.Username).Run()
			}
			ok := err == nil && lingerOn()
			res.Steps = append(res.Steps, Check{Name: "linger", OK: ok,
				Detail: map[bool]string{true: "turned on", false: "could not turn it on"}[ok],
				Fix:    "sudo loginctl enable-linger " + u.Username})
		}
	}
	if a.Linger {
		res.Steps = append(res.Steps, InstallRestoreUnit())
	}
	for name, src := range a.Terminfo {
		if HasTerminfo(name) {
			res.Steps = append(res.Steps, Check{Name: "terminfo " + name, OK: true, Detail: "already installed"})
			continue
		}
		f, err := os.CreateTemp("", "terminfo-*")
		if err == nil {
			_, _ = f.WriteString(src)
			f.Close()
			home, _ := os.UserHomeDir()
			out, terr := exec.Command("tic", "-x", "-o", filepath.Join(home, ".terminfo"), f.Name()).CombinedOutput()
			os.Remove(f.Name())
			err = terr
			if err != nil {
				err = fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
			}
		}
		ok := err == nil && HasTerminfo(name)
		d := "installed"
		if !ok {
			d = fmt.Sprintf("could not install: %v", err)
		}
		res.Steps = append(res.Steps, Check{Name: "terminfo " + name, OK: ok, Detail: d})
	}
	return res
}
