package host

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/iamlakshya/simstim/internal/agents"
	"github.com/iamlakshya/simstim/internal/api"
)

// loginStatusTimeout bounds an agent's own status command.
const loginStatusTimeout = 10 * time.Second

// AgentChecks reports, for each agent the host checks at setup, whether it is
// installed, whether a login shell finds it, and whether it is logged in. An
// agent that is not checked by default is reported when it is installed.
func AgentChecks() []api.AgentCheck {
	login, err := LoginEnv(15 * time.Second)
	if err != nil {
		login = os.Environ()
	}
	home, _ := os.UserHomeDir()
	var list []agents.Agent
	for _, a := range agents.Table {
		if a.Setup != nil {
			list = append(list, a)
		}
	}
	out := make([]api.AgentCheck, len(list))
	var wg sync.WaitGroup
	for i, a := range list {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = checkAgent(a, login, home)
		}()
	}
	wg.Wait()
	kept := out[:0]
	for i, c := range out {
		if list[i].Checked() || c.Installed {
			kept = append(kept, c)
		}
	}
	return kept
}

// checkAgent checks one agent against the login shell's environment.
func checkAgent(a agents.Agent, login []string, home string) api.AgentCheck {
	c := api.AgentCheck{Kind: a.Kind, Title: a.Title, Support: string(a.Support), Login: api.LoginUnknown}
	path := envValue(login, "PATH")
	for _, b := range a.Binaries {
		if p, err := lookPath(b, path); err == nil {
			c.Path, c.OnPath, c.Installed = p, true, true
			break
		}
	}
	if !c.Installed {
		for _, p := range a.Setup.Paths {
			full := agents.HomePath(p, home)
			if st, err := os.Stat(full); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
				c.Path, c.Installed = full, true
				break
			}
		}
	}
	if !c.Installed {
		return c
	}
	c.Login, c.LoginHow = loginState(*a.Setup, c.Path, login, home)
	return c
}

// loginState decides whether an agent is logged in, from the strongest sign
// to the weakest: a variable that logs it in, its own status command, then
// its credential files. It returns the state and how it knows.
func loginState(s agents.Setup, bin string, env []string, home string) (string, string) {
	for _, k := range s.LoginEnv {
		if envValue(env, k) != "" {
			return api.LoginYes, k + " is set in the login shell"
		}
	}
	if len(s.LoginStatus) > 0 {
		// The agent's own answer wins. Any other exit code (an older
		// version without the command, a crash, a timeout) falls through.
		switch runStatus(s.LoginStatus, bin, env) {
		case 0:
			return api.LoginYes, "`" + strings.Join(s.LoginStatus, " ") + "` says so"
		case 1:
			return api.LoginNo, "`" + strings.Join(s.LoginStatus, " ") + "` says not"
		}
	}
	for _, f := range s.LoginFiles {
		if _, err := os.Stat(agents.HomePath(f, home)); err == nil {
			return api.LoginYes, "found " + f
		}
	}
	if len(s.LoginFiles) > 0 {
		return api.LoginNo, "no " + strings.Join(s.LoginFiles, " or ")
	}
	return api.LoginUnknown, "the agent has no status command or credentials file to read"
}

// runStatus runs an agent's status command with the program at bin, and
// returns its exit code, or -1 when it did not run or timed out. Its output
// is thrown away: it can name the account or show part of a key.
func runStatus(argv []string, bin string, env []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), loginStatusTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, argv[1:]...)
	cmd.Args[0] = argv[0]
	cmd.Env = env
	if home, err := os.UserHomeDir(); err == nil {
		cmd.Dir = home
	}
	// Some programs find their own install folder from their path: let them.
	if dir := filepath.Dir(bin); dir != "." {
		cmd.Env = append(append([]string(nil), env...), "PATH="+dir+string(filepath.ListSeparator)+envValue(env, "PATH"))
	}
	err := cmd.Run()
	if ctx.Err() != nil {
		return -1
	}
	var ee *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ee):
		return ee.ExitCode()
	}
	return -1
}
