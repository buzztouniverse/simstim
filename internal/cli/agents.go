package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iAmLakshya/simstim/internal/agents"
	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
)

func agentsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "agents", Short: "List the agents, and install one on a host"}
	cmd.AddCommand(agentsLsCmd(), agentsInstallCmd())
	return cmd
}

func agentsLsCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List the agents in the agent table, with how each one installs",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			if asJSON {
				return printJSON(agents.Table)
			}
			for _, a := range agents.Table {
				install := "—"
				if a.Setup != nil && a.Setup.Install != "" {
					install = a.Setup.Install
				}
				fmt.Printf("%-12s %-16s %-12s %s\n", a.Kind, a.Title, a.Support, install)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

func agentsInstallCmd() *cobra.Command {
	var hostName string
	var force bool
	cmd := &cobra.Command{
		Use:   "install <agent>",
		Short: "Install an agent on a host with its vendor's installer, in a session you can watch",
		Long: "Runs the agent's official install command in a new session on the host, so you see\n" +
			"its output and can answer its questions. Then it says how to log in: you run the\n" +
			"agent's own login, in a session. " + brand.DisplayName + " never logs in for you.",
		Example: "  " + brand.Name + " agents install claude --host my-vps",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, ok := agents.Find(args[0])
			if !ok || a.Setup == nil || a.Setup.Install == "" {
				return &client.UserError{Msg: fmt.Sprintf("there is no install command for %q", args[0]), Hint: brand.Name + " agents ls"}
			}
			h, err := pickHost(hostName)
			if err != nil {
				return err
			}
			return installAgent(cmd.Context(), h, a, force)
		},
	}
	cmd.Flags().StringVar(&hostName, "host", "", "the host (needed when you have more than one)")
	cmd.Flags().BoolVar(&force, "force", false, "run the installer even when the agent is installed, for example to update it")
	return cmd
}

// pickHost is the named host, or the only one.
func pickHost(name string) (client.Host, error) {
	if name != "" {
		return client.FindHost(name)
	}
	hosts, err := client.LoadHosts()
	if err != nil {
		return client.Host{}, err
	}
	switch len(hosts) {
	case 0:
		return client.Host{}, &client.UserError{Msg: "no hosts yet", Hint: brand.Name + " host add <name>"}
	case 1:
		return hosts[0], nil
	}
	names := make([]string, len(hosts))
	for i, h := range hosts {
		names[i] = h.Name
	}
	return client.Host{}, &client.UserError{Msg: "which host? You have " + strings.Join(names, ", "), Hint: "Add --host <name>."}
}

func installAgent(ctx context.Context, h client.Host, a agents.Agent, force bool) error {
	cctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	conn, err := client.Dial(cctx, h, true)
	if err != nil {
		return err
	}
	checks, err := conn.AgentChecks(cctx)
	if err != nil {
		conn.Close()
		return err
	}
	if c, ok := findCheck(checks, a.Kind); ok && c.Installed && !force {
		conn.Close()
		fmt.Printf("%s is already installed on %s, at %s.\n", a.Title, h.Name, c.Path)
		printAgentChecks(os.Stdout, h.Name, []api.AgentCheck{c})
		fmt.Printf("\nTo run its installer again, for example to update it, add --force.\n")
		return nil
	}
	fmt.Printf("Installing %s on %s with its vendor's installer:\n\n    %s\n\n", a.Title, h.Name, a.Setup.Install)
	fmt.Printf("It comes from %s.\nIt runs in a session, so you see its output and can answer its questions.\n\n", a.Setup.Source)
	cols, rows := termSize()
	args := client.TerminalArgs(client.NewArgs{Name: "install-" + a.Kind, Cwd: "~", Argv: []string{"/bin/sh", "-c", a.Setup.Install}, Setup: api.SetupSkip}, cols, rows)
	s, err := conn.NewSession(cctx, args)
	conn.Close()
	if err != nil {
		return err
	}
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		fmt.Printf("Started %s/%s. Attach to answer its questions: %s attach %s/%s\n", h.Name, s.ID, brand.Name, h.Name, s.ID)
		return nil
	}
	ended, err := attachEnded(ctx, h, s.ID, s.Name)
	if err != nil {
		return err
	}
	if !ended {
		fmt.Printf("The installer keeps running. When it is done, run `%s host doctor %s`.\n", brand.Name, h.Name)
		return nil
	}
	return afterInstall(ctx, h, a)
}

// afterInstall checks the agent again and says how to log in.
func afterInstall(ctx context.Context, h client.Host, a agents.Agent) error {
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	conn, err := client.Dial(cctx, h, true)
	if err != nil {
		return err
	}
	defer conn.Close()
	checks, err := conn.AgentChecks(cctx)
	if err != nil {
		return err
	}
	c, ok := findCheck(checks, a.Kind)
	if !ok || !c.Installed {
		return &client.UserError{Msg: fmt.Sprintf("%s is still not installed on %s", a.Title, h.Name),
			Hint: "Read the installer's output above, or see " + a.Setup.Source}
	}
	fmt.Println()
	printAgentChecks(os.Stdout, h.Name, []api.AgentCheck{c})
	if c.Login != api.LoginYes {
		fmt.Printf("\nNext, log in to %s with its own login, in a session on %s:\n\n    %s\n", a.Title, h.Name, loginCommand(h.Name, a))
	}
	return nil
}

func findCheck(checks []api.AgentCheck, kind string) (api.AgentCheck, bool) {
	for _, c := range checks {
		if c.Kind == kind {
			return c, true
		}
	}
	return api.AgentCheck{}, false
}

// loginCommand is the command that starts an agent's own login in a new
// session on host. An agent whose login is its first start starts as itself.
func loginCommand(host string, a agents.Agent) string {
	if a.Setup == nil || a.Setup.Login == "" {
		return ""
	}
	if a.Setup.Login == a.Binaries[0] {
		return fmt.Sprintf("%s new %s --agent %s", brand.Name, host, a.Kind)
	}
	return fmt.Sprintf("%s new %s -- %s", brand.Name, host, a.Setup.Login)
}

// printAgentChecks prints one line for each agent, and what to do next.
func printAgentChecks(w io.Writer, host string, checks []api.AgentCheck) {
	for _, c := range checks {
		a, _ := agents.Find(c.Kind)
		mark, line, try := "✓", "", ""
		switch {
		case !c.Installed:
			mark, line = "✕", "not installed"
			try = fmt.Sprintf("%s agents install %s --host %s", brand.Name, c.Kind, host)
		case !c.OnPath:
			mark, line = "✕", c.Path+" is not on PATH in a login shell"
			try = fmt.Sprintf("on the host, add it to PATH: echo 'export PATH=\"%s:$PATH\"' >> ~/.profile", filepath.Dir(c.Path))
		case c.Login == api.LoginYes:
			line = fmt.Sprintf("%s, logged in (%s)", c.Path, c.LoginHow)
		case c.Login == api.LoginNo:
			mark, line = "!", fmt.Sprintf("%s, not logged in (%s)", c.Path, c.LoginHow)
			try = loginCommand(host, a)
		default:
			mark, line = "?", fmt.Sprintf("%s, login unknown (%s)", c.Path, c.LoginHow)
			if l := loginCommand(host, a); l != "" {
				try = "if it asks you to log in: " + l
			}
		}
		if try == "" && c.Login != api.LoginYes && a.Setup != nil && len(a.Setup.LoginEnv) > 0 && c.OnPath {
			try = fmt.Sprintf("pass its key from your laptop: %s new %s --agent %s --secret %s", brand.Name, host, c.Kind, a.Setup.LoginEnv[0])
		}
		fmt.Fprintf(w, "  %s %-16s %s\n", mark, c.Title, line)
		if try != "" {
			fmt.Fprintf(w, "    Try: %s\n", try)
		}
	}
}
