package api

// Setup values in NewArgs: run the repository's setup script, or skip it.
const (
	SetupRun  = "run"
	SetupSkip = "skip"
)

// CodeSetupTrust is the error code of a new session whose repository has a
// setup script that the user has neither trusted nor refused on this host.
// The error's data is a SetupQuestion.
const CodeSetupTrust = "setup_trust"

// SetupQuestion asks the user whether a repository's setup script may run.
type SetupQuestion struct {
	// Repo is the repository on the host: its main folder, which its
	// worktrees share. The answer is kept by host and Repo.
	Repo string `json:"repo"`
	// Script is the script that would run.
	Script string `json:"script"`
}
