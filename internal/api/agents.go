package api

// Login states of an agent on a host.
const (
	LoginYes     = "yes"
	LoginNo      = "no"
	LoginUnknown = "unknown"
)

// AgentCheck is what a host knows about one agent: is it installed, can a
// login shell find it, and is it logged in.
type AgentCheck struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Support string `json:"support"`
	// Installed is true when the program is on PATH or where its installer puts it.
	Installed bool `json:"installed"`
	// Path is where the program is.
	Path string `json:"path,omitempty"`
	// OnPath is true when a login shell finds the program.
	OnPath bool `json:"on_path"`
	// Login is LoginYes, LoginNo or LoginUnknown, and LoginHow says how the
	// host knows.
	Login    string `json:"login"`
	LoginHow string `json:"login_how,omitempty"`
}
