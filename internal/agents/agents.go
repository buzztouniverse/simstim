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
	// Hooks are extra arguments that make the agent report its status and session
	// ID to the holder. {hook} is replaced by the command that forwards an event.
	Hooks []string `json:"hooks,omitempty"`
	// Support says how much the project promises for this agent.
	Support Support `json:"support"`
}

// Support is how much the project promises for an agent.
type Support string

// The support levels, from most to least.
const (
	// FirstClass agents are tested end to end on every release, report their
	// status through hooks, resume after a reboot, and can be installed with
	// agents install.
	FirstClass Support = "first-class"
	// Supported agents are in the table and tested when the test host has
	// them; their status comes from screen rules.
	Supported Support = "supported"
	// BestEffort agents are in the table, and contributors keep them working.
	BestEffort Support = "best-effort"
)

// claudeHooks makes Claude Code report its session ID, prompts, finished turns
// and notifications, through --settings (merged with the user's own settings).
const claudeHooks = `{"hooks":{` +
	`"SessionStart":[{"hooks":[{"type":"command","command":"{hook} session-start","timeout":5}]}],` +
	`"UserPromptSubmit":[{"hooks":[{"type":"command","command":"{hook} prompt","timeout":5}]}],` +
	`"Stop":[{"hooks":[{"type":"command","command":"{hook} stop","timeout":5}]}],` +
	`"Notification":[{"hooks":[{"type":"command","command":"{hook} notification","timeout":5}]}]}}`

// Table is the built-in agent table, checked against each agent's docs in
// October 2026. First-class agents come first: the launcher lists them in
// this order.
var Table = []Agent{
	{
		Kind: "claude", Title: "Claude Code", Binaries: []string{"claude"},
		Launch:    []string{"claude", "--session-id", "{session_uuid}"},
		Resume:    []string{"claude", "--resume", "{agent_session_id}"},
		Continue:  []string{"claude", "--continue"},
		IDCapture: "choose",
		Hooks:     []string{"--settings", claudeHooks},
		Support:   FirstClass,
	},
	{
		Kind: "codex", Title: "Codex", Binaries: []string{"codex"},
		// --no-daemon keeps Codex's work inside the session's PTY and cgroup.
		Launch:    []string{"codex", "--no-daemon"},
		Resume:    []string{"codex", "--no-daemon", "resume", "{agent_session_id}"},
		Continue:  []string{"codex", "--no-daemon", "resume", "--last"},
		IDCapture: "hook",
		// Codex runs the notify program after each turn, with a JSON argument.
		Hooks:   []string{"-c", `notify=["{hook_bin}","hook","codex-notify"]`},
		Support: FirstClass,
	},
	{
		Kind: "opencode", Title: "opencode", Binaries: []string{"opencode"},
		Launch:    []string{"opencode"},
		Resume:    []string{"opencode", "-s", "{agent_session_id}"},
		Continue:  []string{"opencode", "-c"},
		IDCapture: "scan",
		Support:   Supported,
	},
	{
		// Antigravity CLI replaced Gemini CLI for Google's consumer plans on
		// 18 June 2026 (https://www.antigravity.google/docs/cli/commands/resume).
		Kind: "antigravity", Title: "Antigravity CLI", Binaries: []string{"agy"},
		Launch:    []string{"agy"},
		Resume:    []string{"agy", "--conversation", "{agent_session_id}"},
		Continue:  []string{"agy", "--continue"},
		IDCapture: "none",
		Support:   Supported,
	},
	{
		Kind: "gemini", Title: "Gemini CLI", Binaries: []string{"gemini"},
		Launch:    []string{"gemini", "--session-id", "{session_uuid}"},
		Resume:    []string{"gemini", "--resume", "{agent_session_id}"},
		Continue:  []string{"gemini", "--resume"},
		IDCapture: "choose",
		Support:   BestEffort, // consumer plans moved to Antigravity CLI
	},
	{
		Kind: "amp", Title: "Amp", Binaries: []string{"amp"},
		Launch:    []string{"amp"},
		Resume:    []string{"amp", "threads", "continue", "{agent_session_id}"},
		Continue:  []string{"amp", "threads", "continue", "--last"},
		IDCapture: "scan",
		Support:   BestEffort,
	},
	{
		Kind: "cursor", Title: "Cursor CLI", Binaries: []string{"agent", "cursor-agent"},
		Launch:    []string{"agent"},
		Resume:    []string{"agent", "--resume", "{agent_session_id}"},
		Continue:  []string{"agent", "--continue"},
		IDCapture: "scan",
		Support:   BestEffort,
	},
	{
		Kind: "aider", Title: "Aider", Binaries: []string{"aider"},
		Launch:    []string{"aider"},
		Continue:  []string{"aider", "--restore-chat-history"},
		IDCapture: "none",
		Support:   BestEffort,
	},
	{
		Kind: "goose", Title: "Goose", Binaries: []string{"goose"},
		Launch:    []string{"goose", "session"},
		Continue:  []string{"goose", "session", "-r"},
		IDCapture: "none",
		Support:   BestEffort,
	},
	{
		Kind: "shell", Title: "Shell", Binaries: []string{"$SHELL"},
		Launch:    []string{"$SHELL", "-l"},
		Continue:  []string{"$SHELL", "-l"},
		IDCapture: "none",
		Support:   Supported, // and through it, any program
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
