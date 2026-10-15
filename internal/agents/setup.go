package agents

import (
	"path/filepath"
	"strings"
)

// Setup is what the project knows about installing an agent on a host and
// logging in to it. Each command comes from the vendor's own docs or
// repository; Source links to it, and the table's comments say where each
// login detail comes from.
type Setup struct {
	// Install is the vendor's install command, a shell command line. It runs
	// in a session on the host, so its output shows and it can ask questions.
	Install string `json:"install,omitempty"`
	// Source is the vendor page that gives the install command.
	Source string `json:"source,omitempty"`
	// Paths are where the installer puts the program ("~/.local/bin/claude"),
	// so a check can tell "not on PATH" from "not installed".
	Paths []string `json:"paths,omitempty"`
	// Login is the command that starts the agent's own login. The user runs
	// it in a session; nothing runs it for them.
	Login string `json:"login,omitempty"`
	// LoginStatus is a command that exits 0 when the agent is logged in and 1
	// when it is not. Its output is never read or shown.
	LoginStatus []string `json:"login_status,omitempty"`
	// LoginFiles are credential files ("~/.codex/auth.json"); any one of them
	// means the agent is logged in.
	LoginFiles []string `json:"login_files,omitempty"`
	// LoginEnv are environment variables that log the agent in when set, for
	// example an API key passed as a session secret.
	LoginEnv []string `json:"login_env,omitempty"`
}

// Installable lists the agents that have an install command, in table order.
func Installable() []Agent {
	var out []Agent
	for _, a := range Table {
		if a.Setup != nil && a.Setup.Install != "" {
			out = append(out, a)
		}
	}
	return out
}

// HomePath expands a leading "~/" against home.
func HomePath(p, home string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return filepath.Join(home, rest)
	}
	return p
}

// Checked reports whether hosts check an agent at setup: every first-class
// and supported agent, and any other one that is installed.
func (a Agent) Checked() bool {
	return a.Setup != nil && (a.Support == FirstClass || a.Support == Supported)
}
