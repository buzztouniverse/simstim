package host

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/iAmLakshya/simstim/internal/paths"
)

// Session states, shown in the client's session list.
const (
	StateStarting   = "starting"
	StateWorking    = "working"
	StateIdle       = "idle"
	StateNeedsInput = "needs_input"
	StateExited     = "exited"
	StateErrored    = "errored"
	StateLost       = "lost" // the holder is gone, for example after a reboot
)

// Manifest is the record of one session, at ~/.local/state/<name>/sessions/<id>/manifest.json.
// Readers keep fields they do not know, so v2 can add fields freely.
type Manifest struct {
	Schema         int       `json:"schema"`
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastActivityAt time.Time `json:"last_activity_at,omitempty"`
	LastAttachedAt time.Time `json:"last_attached_at,omitempty"`
	HostVersion    string    `json:"host_version"`
	HolderBinary   string    `json:"holder_binary"`

	Agent  AgentInfo         `json:"agent"`
	Cwd    string            `json:"cwd"`
	Env    []string          `json:"env"`
	Term   TermInfo          `json:"term"`
	Limits Limits            `json:"limits"`
	Run    Runtime           `json:"runtime"`
	Status Status            `json:"status"`
	Labels map[string]string `json:"labels,omitempty"`

	// Restart is used by v2 respawn.
	Restart Restart `json:"restart"`

	extra map[string]json.RawMessage
}

// AgentInfo is the agent part of a manifest.
type AgentInfo struct {
	Kind           string   `json:"kind"`
	Argv           []string `json:"argv"`
	ResumeArgv     []string `json:"resume_argv,omitempty"`
	ContinueArgv   []string `json:"continue_argv,omitempty"`
	AgentSessionID string   `json:"agent_session_id,omitempty"`
	IDSource       string   `json:"id_source,omitempty"`
}

// TermInfo is the terminal identity given to the agent.
type TermInfo struct {
	Term string `json:"TERM"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// Limits are the resource caps of the session.
type Limits struct {
	MemoryHigh int64 `json:"memory_high,omitempty"`
	MemoryMax  int64 `json:"memory_max,omitempty"`
}

// Runtime is live process information.
type Runtime struct {
	HolderPID    int       `json:"holder_pid,omitempty"`
	AgentPID     int       `json:"agent_pid,omitempty"`
	Unit         string    `json:"unit,omitempty"`
	Socket       string    `json:"socket"`
	BootID       string    `json:"boot_id,omitempty"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	RespawnCount int       `json:"respawn_count"`
}

// Status is the session's state and why.
type Status struct {
	State    string    `json:"state"`
	Since    time.Time `json:"since"`
	Source   string    `json:"source,omitempty"`
	ExitCode *int      `json:"exit_code,omitempty"`
	Signal   string    `json:"signal,omitempty"`
	OOM      bool      `json:"oom_killed,omitempty"`
	Reason   string    `json:"reason,omitempty"`
}

// Restart is the respawn policy (v2).
type Restart struct {
	Policy      string `json:"policy"`
	MaxAttempts int    `json:"max_attempts"`
}

// UnmarshalJSON keeps unknown fields.
func (m *Manifest) UnmarshalJSON(b []byte) error {
	type plain Manifest
	if err := json.Unmarshal(b, (*plain)(m)); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(b, &all); err != nil {
		return err
	}
	known := map[string]bool{"schema": true, "id": true, "name": true, "created_at": true, "updated_at": true,
		"last_activity_at": true, "last_attached_at": true, "host_version": true, "holder_binary": true,
		"agent": true, "cwd": true, "env": true, "term": true, "limits": true, "runtime": true, "status": true,
		"labels": true, "restart": true}
	for k := range all {
		if known[k] {
			delete(all, k)
		}
	}
	m.extra = all
	return nil
}

// MarshalJSON writes unknown fields back.
func (m Manifest) MarshalJSON() ([]byte, error) {
	type plain Manifest
	b, err := json.Marshal(plain(m))
	if err != nil || len(m.extra) == 0 {
		return b, err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(b, &all); err != nil {
		return nil, err
	}
	for k, v := range m.extra {
		all[k] = v
	}
	return json.Marshal(all)
}

// LoadManifest reads one session's manifest.
func LoadManifest(id string) (*Manifest, error) {
	b, err := os.ReadFile(paths.Manifest(id))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Save writes the manifest atomically.
func (m *Manifest) Save() error {
	m.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	dir := paths.SessionDir(m.ID)
	if err := paths.EnsurePrivateDir(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".manifest-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	tmp.Close()
	return os.Rename(tmp.Name(), filepath.Join(dir, "manifest.json"))
}

// BootID identifies the current boot, so a respawn can tell that a reboot happened.
func BootID() string {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ""
	}
	return string(trimNL(b))
}

func trimNL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}
