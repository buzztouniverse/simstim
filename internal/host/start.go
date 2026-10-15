package host

import (
	"fmt"
	"strings"

	"github.com/iAmLakshya/simstim/internal/brand"
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
