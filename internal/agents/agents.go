// Package agents is the product's knowledge of each agent CLI, kept as data.
//
// Nothing else in the code base branches on an agent's name. To support a new agent,
// add a row here.
package agents

import "strings"

// Agent describes one agent CLI.
type Agent struct {
	Kind     string   `json:"kind"`
	Title    string   `json:"title"`
	Binaries []string `json:"binaries"`
	// Launch is the argv to start a new session. {session_uuid} is replaced.
	Launch []string `json:"launch"`
	// Resume and Continue are used to respawn after a host reboot (v2).
	// {agent_session_id} is replaced.
	Resume   []string `json:"resume,omitempty"`
	Continue []string `json:"continue,omitempty"`
	// IDCapture says how we learn the agent's own session ID:
	// "choose" (we pass it), "hook", "scan" or "none".
	IDCapture string `json:"id_capture"`
}

// Table is the built-in agent table, checked against each agent's docs in October 2026.
var Table = []Agent{
	{
		Kind: "claude", Title: "Claude Code", Binaries: []string{"claude"},
		Launch:    []string{"claude", "--session-id", "{session_uuid}"},
		Resume:    []string{"claude", "--resume", "{agent_session_id}"},
		Continue:  []string{"claude", "--continue"},
		IDCapture: "choose",
	},
	{
		Kind: "codex", Title: "Codex", Binaries: []string{"codex"},
		// --no-daemon keeps Codex's work inside the session's PTY and cgroup.
		Launch:    []string{"codex", "--no-daemon"},
		Resume:    []string{"codex", "--no-daemon", "resume", "{agent_session_id}"},
		Continue:  []string{"codex", "--no-daemon", "resume", "--last"},
		IDCapture: "hook",
	},
	{
		Kind: "opencode", Title: "opencode", Binaries: []string{"opencode"},
		Launch:    []string{"opencode"},
		Resume:    []string{"opencode", "-s", "{agent_session_id}"},
		Continue:  []string{"opencode", "-c"},
		IDCapture: "scan",
	},
	{
		Kind: "gemini", Title: "Gemini CLI", Binaries: []string{"gemini"},
		Launch:    []string{"gemini", "--session-id", "{session_uuid}"},
		Resume:    []string{"gemini", "--resume", "{agent_session_id}"},
		Continue:  []string{"gemini", "--resume"},
		IDCapture: "choose",
	},
	{
		Kind: "amp", Title: "Amp", Binaries: []string{"amp"},
		Launch:    []string{"amp"},
		Resume:    []string{"amp", "threads", "continue", "{agent_session_id}"},
		Continue:  []string{"amp", "threads", "continue", "--last"},
		IDCapture: "scan",
	},
	{
		Kind: "cursor", Title: "Cursor CLI", Binaries: []string{"agent", "cursor-agent"},
		Launch:    []string{"agent"},
		Resume:    []string{"agent", "--resume", "{agent_session_id}"},
		Continue:  []string{"agent", "--continue"},
		IDCapture: "scan",
	},
	{
		Kind: "aider", Title: "Aider", Binaries: []string{"aider"},
		Launch:    []string{"aider"},
		Continue:  []string{"aider", "--restore-chat-history"},
		IDCapture: "none",
	},
	{
		Kind: "goose", Title: "Goose", Binaries: []string{"goose"},
		Launch:    []string{"goose", "session"},
		Continue:  []string{"goose", "session", "-r"},
		IDCapture: "none",
	},
	{
		Kind: "shell", Title: "Shell", Binaries: []string{"$SHELL"},
		Launch:    []string{"$SHELL", "-l"},
		Continue:  []string{"$SHELL", "-l"},
		IDCapture: "none",
	},
}

// Find returns the agent with the given kind.
func Find(kind string) (Agent, bool) {
	for _, a := range Table {
		if a.Kind == kind {
			return a, true
		}
	}
	return Agent{}, false
}

// Expand replaces placeholders in argv.
func Expand(argv []string, vars map[string]string) []string {
	out := make([]string, len(argv))
	for i, a := range argv {
		for k, v := range vars {
			a = strings.ReplaceAll(a, "{"+k+"}", v)
		}
		out[i] = a
	}
	return out
}
