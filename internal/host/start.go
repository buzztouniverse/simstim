package host

import (
	"fmt"
	"strings"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// startArgv wraps an agent's argv in a short sh script when something must
// show in the session's terminal before the agent: notes, then a hook, whose
// output is part of the session. A failing hook is reported, not fatal. The
// script then execs the agent, so the agent is still the session's process.
func startArgv(argv []string, hook string, notes []string) []string {
	if hook == "" && len(notes) == 0 {
		return argv
	}
	var b strings.Builder
	for _, n := range notes {
		fmt.Fprintf(&b, "printf '%%s\\n\\n' %s; ", shellQuote(n))
	}
	label := brand.Name
	if hook != "" {
		b.WriteString(`"$0"; s=$?; [ $s -eq 0 ] || printf '\n[hook %s exited with %s]\n\n' "$0" $s; `)
		label = hook
	}
	b.WriteString(`exec "$@"`)
	return append([]string{"/bin/sh", "-c", b.String(), label}, argv...)
}
