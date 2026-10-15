package onboard

import (
	"fmt"
	"path"

	"github.com/iamlakshya/simstim/internal/agents"
	"github.com/iamlakshya/simstim/internal/api"
	"github.com/iamlakshya/simstim/internal/brand"
)

// Line is one agent as a summary shows it: a mark, its title, what is true,
// and what to try next.
type Line struct {
	// Mark is ✓ ready, ! not logged in, ? login unknown, or ✕ not ready.
	Mark  string
	Title string
	// State says what is true, with the path and how the host knows; Short
	// says it in a few words.
	State string
	Short string
	Try   string
}

// Describe says where an agent stands on host, from the host's check, and
// what to do next.
func Describe(host string, c api.AgentCheck) Line {
	a, _ := agents.Find(c.Kind)
	l := Line{Mark: "✓", Title: c.Title}
	switch {
	case !c.Installed:
		l.Mark, l.State, l.Short = "✕", "not installed", "not installed"
		l.Try = fmt.Sprintf("%s agents install %s --host %s", brand.Name, c.Kind, host)
	case !c.OnPath:
		l.Mark, l.State, l.Short = "✕", c.Path+" is not on PATH in a login shell", "installed, not on PATH"
		l.Try = fmt.Sprintf("on the host, add it to PATH: echo 'export PATH=\"%s:$PATH\"' >> ~/.profile", path.Dir(c.Path))
	case c.Login == api.LoginYes:
		l.State, l.Short = fmt.Sprintf("%s, logged in (%s)", c.Path, c.LoginHow), "installed, logged in"
	case c.Login == api.LoginNo:
		l.Mark, l.State, l.Short = "!", fmt.Sprintf("%s, not logged in (%s)", c.Path, c.LoginHow), "installed, not logged in"
		l.Try = LoginCommand(host, a)
	default:
		l.Mark, l.State, l.Short = "?", fmt.Sprintf("%s, login unknown (%s)", c.Path, c.LoginHow), "installed, login unknown"
		if cmd := LoginCommand(host, a); cmd != "" {
			l.Try = "if it asks you to log in: " + cmd
		}
	}
	if l.Try == "" && c.Login != api.LoginYes && a.Setup != nil && len(a.Setup.LoginEnv) > 0 && c.OnPath {
		l.Try = fmt.Sprintf("pass its key from your laptop: %s new %s --agent %s --secret %s", brand.Name, host, c.Kind, a.Setup.LoginEnv[0])
	}
	return l
}
