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
	"path/filepath"
	"sort"
	"strconv"
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
	secrets, err := checkSecrets(a.Secrets)
	if err != nil {
		return nil, err
	}
	login, err := LoginEnv(15 * time.Second)
	if err != nil {
		login = os.Environ()
	}
	id := NewID()
	var gi *GitInfo
	if a.Worktree {
		n := a.Name
		if strings.TrimSpace(n) == "" {
			n = filepath.Base(cwd)
		}
		gi, err = makeWorktree(cwd, n, id, a.Base)
		if err != nil {
			return nil, err
		}
		cwd = gi.Worktree
	}
	sessHooks, err := setupHooks(a, cwd, gi)
	if err != nil {
		return nil, err
	}
	term := a.Term
	if !HasTerminfo(term) {
		term = "xterm-256color"
	}
	colorterm := a.ColorTerm
	if colorterm == "" {
		colorterm = "truecolor"
	}
	extra := map[string]string{
		"TERM": term, "COLORTERM": colorterm,
		"TERM_PROGRAM": a.TermProgram, "TERM_PROGRAM_VERSION": a.TermProgramVersion,
		brand.Env("SESSION"): id, brand.Env("SOCK"): paths.Socket(id),
	}
	inline := a.Inline && known && len(a.Argv) == 0 && agent.Inline.Has()
	if inline {
		for k, v := range agent.Inline.Env {
			extra[k] = v
		}
	}
	env := AgentEnv(login, extra)
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
	if inline {
		argv = append(argv, agent.Inline.Args...)
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
	m.Git = gi
	m.Secrets = secrets
	m.SetupHook, m.ResumeHook = sessHooks.setup, sessHooks.resume
	m.RunHook = m.SetupHook
	if known {
		m.Agent.ResumeArgv = agent.Resume
		m.Agent.ContinueArgv = agent.Continue
		m.Agent.Hooks = hooks
		if inline {
			m.Agent.Inline = agent.Inline.Args
		}
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
	if err := deliverSecrets(id, a.Secrets); err != nil {
		_ = deleteSession(id, &DeleteResult{})
		return nil, UserError("secrets_failed", "the session could not get its secrets, so it was removed", "Try again.")
	}
	if mm := waitStarted(id, 5*time.Second); mm != nil {
		if mm.Run.Unit == "" && m.Run.Unit != "" {
			mm.Run.Unit = m.Run.Unit
			_ = mm.Save()
		}
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
		State: st.State, Source: st.Source, Since: st.Since, Reason: st.Reason, ExitCode: st.ExitCode, Signal: st.Signal, OOM: st.OOM,
		CreatedAt: m.CreatedAt, LastActivityAt: m.LastActivityAt, LastAttachedAt: m.LastAttachedAt,
		Alive: alive, Unit: m.Run.Unit, MemoryMax: m.Limits.MemoryMax, MemoryCurrent: memoryCurrent(m.Run.Unit),
		HostVersion: m.HostVersion, Cols: m.Term.Cols, Rows: m.Term.Rows, MissingSecrets: m.MissingSecrets,
		FullScreen: st.FullScreen && alive,
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
	n, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
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

// Delete stops a session and removes its files, and its worktree when that loses nothing.
func Delete(id string) (DeleteResult, error) {
	var res DeleteResult
	err := deleteSession(id, &res)
	return res, err
}

// DeleteResult says what deleting a session did.
type DeleteResult = api.DeleteResult

func deleteSession(id string, res *DeleteResult) error {
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
	if m != nil {
		res.Worktree = removeWorktree(m.Git)
	}
	if err := os.RemoveAll(paths.SessionDir(id)); err != nil {
		return err
	}
	res.Deleted = true
	return nil
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
