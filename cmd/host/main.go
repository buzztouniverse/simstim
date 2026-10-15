// Command host is the helper the client installs on each host. It keeps sessions
// alive and serves the client over the SSH channel's stdio.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/version"
)

const usage = `%[1]s keeps agent sessions alive on this host. The client runs it over SSH.

Usage:
  %[1]s bridge              serve one client on stdin and stdout
  %[1]s ls [--json]         list sessions
  %[1]s new [--agent a] [--dir d] [--name n] [-- argv…]
  %[1]s rm <id>             stop a session and remove it
  %[1]s doctor [--json]     check this host
  %[1]s keytest             print every key the terminal sends
  %[1]s restore [--all]     start again sessions a restart (or, with --all, a crash) stopped
  %[1]s version
`

func main() {
	name := brand.HostBinary
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, usage, name)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "bridge":
		err = host.RunBridge(os.Stdin, os.Stdout)
	case "holder":
		if len(os.Args) != 3 {
			err = fmt.Errorf("usage: %s holder <id>", name)
			break
		}
		err = host.RunHolder(os.Args[2])
	case "ls":
		var list []host.SessionInfo
		list, err = host.List()
		if err == nil {
			if hasFlag("--json") {
				err = printJSON(list)
			} else {
				for _, s := range list {
					fmt.Printf("%-10s %-20s %-9s %-12s %s\n", s.ID, s.Name, s.Agent, s.State, s.Cwd)
				}
			}
		}
	case "new":
		a := host.NewArgs{Term: os.Getenv("TERM"), ColorTerm: os.Getenv("COLORTERM"),
			TermProgram: os.Getenv("TERM_PROGRAM"), TermProgramVersion: os.Getenv("TERM_PROGRAM_VERSION")}
		args := os.Args[2:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--agent":
				i++
				a.Agent = arg(args, i)
			case "--dir":
				i++
				a.Cwd = arg(args, i)
			case "--name":
				i++
				a.Name = arg(args, i)
			case "--":
				a.Argv = args[i+1:]
				i = len(args)
			}
		}
		var info *host.SessionInfo
		info, err = host.New(a)
		if err == nil {
			err = printJSON(info)
		}
	case "rm":
		if len(os.Args) != 3 {
			err = fmt.Errorf("usage: %s rm <id>", name)
			break
		}
		var res host.DeleteResult
		res, err = host.Delete(os.Args[2])
		if err == nil && res.Worktree != "" {
			fmt.Println(res.Worktree)
		}
	case "doctor":
		d := host.Doctor()
		if hasFlag("--json") {
			err = printJSON(d)
		} else {
			fmt.Printf("%s %s on %s (%s, %s/%s)\n", name, d.Version, d.Hostname, d.Distro, d.OS, d.Arch)
			for _, c := range d.Checks {
				mark := "✓"
				if !c.OK {
					mark = "✕"
				}
				fmt.Printf("  %s %-22s %s\n", mark, c.Name, c.Detail)
				if !c.OK && c.Fix != "" {
					fmt.Printf("    Try: %s\n", c.Fix)
				}
			}
			var found []string
			for k := range d.Agents {
				found = append(found, k)
			}
			fmt.Printf("  agents: %s\n", strings.Join(found, ", "))
		}
	case "hook":
		if len(os.Args) >= 3 {
			host.RunHook(os.Args[2], os.Args[3:])
		}
		return
	case "restore":
		res := host.Restore(hasFlag("--all"))
		if hasFlag("--json") {
			err = printJSON(res)
		} else {
			for _, r := range res {
				fmt.Printf("%-10s %-20s %-9s %s\n", r.ID, r.Name, r.Action, r.Why)
			}
		}
	case "keytest":
		err = host.KeyTest()
	case "pick":
		// The phone path: experimental, and left out of the usage on purpose
		// until it is launched (docs/contributing/phone-path.md).
		err = host.RunPick(os.Stdin, os.Stdout)
	case "version", "--version":
		fmt.Printf("%s %s (protocol %s)\n", name, version.Version, version.Protocol)
	case "help", "-h", "--help":
		fmt.Printf(usage, name)
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func arg(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

func hasFlag(f string) bool {
	for _, a := range os.Args[2:] {
		if a == f {
			return true
		}
	}
	return false
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
