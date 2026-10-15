package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/client/helpers"
)

func hostCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "host", Short: "Add, list, test and remove hosts"}
	cmd.AddCommand(hostAddCmd(), hostLsCmd(), hostRmCmd(), hostTestCmd(), hostDoctorCmd(), hostRestoreCmd(), hostAlertsCmd(),
		hostProviderCmd("start", "Start the host through its provider script"),
		hostProviderCmd("stop", "Stop the host through its provider script (its sessions stop too)"),
		hostProviderCmd("status", "Ask the provider script whether the host is running"))
	return cmd
}

func hostAddCmd() *cobra.Command {
	var sshConfig, label, providerScript string
	var yes, wake bool
	cmd := &cobra.Command{
		Use:   "add <name> [ssh-target]",
		Short: "Connect a host: check it, install the helper, prepare it",
		Long: "Connect a host. The ssh target is a Host from your SSH config, or user@host.\n" +
			"It defaults to the name. --ssh-config adds an extra SSH config file, for example a repo's own.",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			h := client.Host{Name: args[0], SSH: args[0], SSHConfig: sshConfig}
			if providerScript != "" {
				h.Provider = &client.Provider{Script: providerScript, WakeOnAttach: wake}
			}
			if len(args) == 2 {
				h.SSH = args[1]
			}
			if label != "" {
				h.Labels = map[string]string{}
				for _, kv := range strings.Split(label, ",") {
					k, v, _ := strings.Cut(kv, "=")
					h.Labels[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
			}
			if !client.ValidHostName(h.Name) {
				return &client.UserError{Msg: fmt.Sprintf("%q cannot name a host", h.Name), Hint: "Use letters, digits, dots, dashes and underscores."}
			}
			return addHost(cmd.Context(), h, yes)
		},
	}
	cmd.Flags().StringVar(&sshConfig, "ssh-config", "", "an extra SSH config file to include")
	cmd.Flags().StringVar(&label, "label", "", "labels, for example provider=gcp,region=eu")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "install the helper without asking")
	cmd.Flags().StringVar(&providerScript, "provider-script", "", "a script that starts and stops the host (see examples/providers)")
	cmd.Flags().BoolVar(&wake, "wake", false, "start the host through its provider script when it does not answer")
	return cmd
}

func addHost(ctx context.Context, h client.Host, yes bool) error {
	s := &client.SSH{Host: h, Interactive: true}
	step := func(format string, a ...any) { fmt.Printf("  "+format+"\n", a...) }
	fmt.Printf("Connecting to %s (%s)…\n", h.Name, h.SSH)
	start := time.Now()
	pctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	p, err := client.Probe(pctx, s)
	cancel()
	if err != nil {
		return err
	}
	step("✓ connected in %s: %s/%s", time.Since(start).Round(time.Millisecond), p.OS, p.Arch)
	if !p.Installed {
		bin, ok := helpers.For(p.OS, p.Arch)
		if !ok {
			return &client.UserError{Msg: fmt.Sprintf("no helper for %s/%s in this build", p.OS, p.Arch), Hint: "Linux hosts on amd64 and arm64 are supported."}
		}
		fmt.Printf("\n%s needs a small helper on the host to keep sessions alive.\n", brand.DisplayName)
		fmt.Printf("    What   %s %s, %.1f MB, SHA-256 checked\n", brand.HostBinary, client.HelperVersion(), float64(len(bin))/1e6)
		fmt.Printf("    Where  ~/.local/share/%s/bin/%s/\n", brand.Name, client.HelperVersion())
		fmt.Printf("    Also   turn on linger, install this terminal's terminfo\n")
		fmt.Printf("  Nothing listens on the network. To remove it later, delete ~/.local/share/%s on the host.\n\n", brand.Name)
		if !yes && !confirm("Install it?") {
			return &client.UserError{Msg: "cancelled; nothing was installed", Hint: brand.Name + " host add " + h.Name}
		}
		ictx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		err := client.Install(ictx, s, p)
		cancel()
		if err != nil {
			return err
		}
		step("✓ installed %s %s", brand.HostBinary, client.HelperVersion())
	} else {
		step("✓ %s %s is already installed", brand.HostBinary, client.HelperVersion())
	}
	dctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	conn, err := client.Dial(dctx, h, true)
	if err != nil {
		return err
	}
	defer conn.Close()
	res, err := conn.Setup(dctx)
	if err != nil {
		return err
	}
	for _, s := range res.Steps {
		mark := "✓"
		if !s.OK {
			mark = "✕"
		}
		step("%s %s: %s", mark, s.Name, s.Detail)
		if !s.OK && s.Fix != "" {
			step("  Try: %s", s.Fix)
		}
	}
	d, err := conn.Doctor(dctx)
	if err != nil {
		return err
	}
	var agents []string
	for k := range d.Agents {
		agents = append(agents, k)
	}
	sort.Strings(agents)
	step("✓ %s, %d CPUs, %.1f GiB; %.1f GiB per session", d.Distro, d.CPUs, float64(d.MemTotal)/(1<<30), float64(d.MemoryMax)/(1<<30))
	step("✓ agents found: %s", strings.Join(agents, ", "))
	hosts, err := client.LoadHosts()
	if err != nil {
		return err
	}
	replaced := false
	for i := range hosts {
		if hosts[i].Name == h.Name {
			hosts[i], replaced = h, true
		}
	}
	if !replaced {
		hosts = append(hosts, h)
	}
	if err := client.SaveHosts(hosts); err != nil {
		return err
	}
	fmt.Printf("\n%s is ready. Start a session with `%s new %s --agent claude`, or run `%s`.\n", h.Name, brand.Name, h.Name, brand.Name)
	return nil
}

func confirm(q string) bool {
	fmt.Printf("%s [y/N] ", q)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

func hostLsCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List hosts",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			hosts, err := client.LoadHosts()
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(hosts)
			}
			if len(hosts) == 0 {
				fmt.Printf("No hosts yet. Add one with `%s host add <name>`.\n", brand.Name)
				return nil
			}
			for _, h := range hosts {
				extra := ""
				if h.SSHConfig != "" {
					extra = "  (+ " + h.SSHConfig + ")"
				}
				fmt.Printf("%-16s %s%s\n", h.Name, h.SSH, extra)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

func hostRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Forget a host (its sessions keep running)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hosts, err := client.LoadHosts()
			if err != nil {
				return err
			}
			out := hosts[:0]
			found := false
			for _, h := range hosts {
				if h.Name == args[0] {
					found = true
					continue
				}
				out = append(out, h)
			}
			if !found {
				return &client.UserError{Msg: fmt.Sprintf("no host named %q", args[0]), Hint: brand.Name + " host ls"}
			}
			if err := client.SaveHosts(out); err != nil {
				return err
			}
			fmt.Printf("Forgot %s. Its sessions and the helper stay on the host.\n", args[0])
			return nil
		},
	}
}

func hostTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <name>",
		Short: "Check that a host answers, and how fast",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := client.FindHost(args[0])
			if err != nil {
				return err
			}
			start := time.Now()
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			conn, err := client.Dial(ctx, h, true)
			if err != nil {
				return err
			}
			defer conn.Close()
			connect := time.Since(start)
			rt, err := conn.RoundTrip(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("%s answers: connect %s, round trip %s\n", h.Name, connect.Round(time.Millisecond), rt.Round(time.Millisecond))
			return nil
		},
	}
}

func hostDoctorCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "doctor <name>",
		Short: "Check a host's setup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := client.FindHost(args[0])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()
			conn, err := client.Dial(ctx, h, true)
			if err != nil {
				return err
			}
			defer conn.Close()
			d, err := conn.Doctor(ctx)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(d)
			}
			fmt.Printf("%s: %s (%s/%s), %d CPUs, %.1f GiB, %.1f GiB free disk, helper %s\n", h.Name, d.Distro, d.OS, d.Arch, d.CPUs,
				float64(d.MemTotal)/(1<<30), float64(d.DiskFree)/(1<<30), d.Version)
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
			var agents []string
			for k, p := range d.Agents {
				agents = append(agents, k+" ("+p+")")
			}
			sort.Strings(agents)
			fmt.Printf("  agents: %s\n", strings.Join(agents, ", "))
			printScrollback(ctx, conn)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func hostRestoreCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "restore <name>",
		Short: "Start again the sessions a host restart stopped (the host also does this at boot)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := client.FindHost(args[0])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			conn, err := client.Dial(ctx, h, true)
			if err != nil {
				return err
			}
			defer conn.Close()
			res, err := conn.Restore(ctx, all)
			if err != nil {
				return err
			}
			if len(res) == 0 {
				fmt.Printf("Nothing to restore on %s.\n", h.Name)
			}
			for _, r := range res {
				fmt.Printf("%-10s %-20s %-9s %s\n", r.ID, r.Name, r.Action, r.Why)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "also restart sessions whose holder crashed without a host restart")
	return cmd
}

func hostProviderCmd(action, short string) *cobra.Command {
	return &cobra.Command{
		Use:   action + " <name>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := client.FindHost(args[0])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer cancel()
			out, err := client.RunProvider(ctx, h, action)
			if err != nil {
				return err
			}
			if s := strings.TrimSpace(string(out)); s != "" {
				fmt.Println(s)
			} else {
				fmt.Printf("%s: %s done.\n", h.Name, action)
			}
			return nil
		},
	}
}
