package host

import (
	"fmt"
	"strings"
	"time"

	"github.com/iamlakshya/simstim/internal/brand"
)

// startArgv wraps an agent's argv in a short sh script when something must
// show in the session's terminal before the agent: notes, then a hook, which
// gets vars in its environment and whose output is part of the session. A
// failing hook is reported with its exit code, and the agent starts anyway.
// The script then execs the agent, so the agent is the session's process.
func startArgv(argv []string, hook string, vars, notes []string) []string {
	if hook == "" && len(notes) == 0 {
		return argv
	}
	var b strings.Builder
	for _, n := range notes {
		fmt.Fprintf(&b, "printf '%%s\\n\\n' %s; ", shellQuote(n))
	}
	label := brand.Name
	if hook != "" {
		name := shellQuote(brand.Name)
		fmt.Fprintf(&b, `printf '[%%s: running %%s]\n' %s "$0"; `, name)
		if len(vars) > 0 {
			b.WriteString(quoteVars(vars) + " ")
		}
		fmt.Fprintf(&b, `"$0"; s=$?; [ $s -eq 0 ] || printf '\n[%%s: %%s exited with %%s; the agent starts anyway]\n\n' %s "$0" $s; `, name)
		label = hook
	}
	b.WriteString(`exec "$@"`)
	return append([]string{"/bin/sh", "-c", b.String(), label}, argv...)
}

// waitStarted waits until the holder has recorded its agent's start (or
// failure) in the manifest, and returns that manifest. The holder writes it
// after its socket answers, and later still when it first waits for secrets,
// so reading it at once would see, and could save over, the record of a
// session that has not started.
func waitStarted(id string, timeout time.Duration) *Manifest {
	deadline := time.Now().Add(timeout)
	for {
		m, err := LoadManifest(id)
		if err == nil && (m.Run.AgentPID != 0 || m.Status.State != StateStarting) {
			return m
		}
		if time.Now().After(deadline) {
			return m
		}
		time.Sleep(20 * time.Millisecond)
	}
}
