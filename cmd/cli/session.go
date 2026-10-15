package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
)

func isTerminal(f *os.File) bool { return term.IsTerminal(int(f.Fd())) }

func termSize() (int, int) {
	c, r, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return c, r
}

func newCmd() *cobra.Command {
	var agent, dir, name string
	var detach bool
	cmd := &cobra.Command{
		Use:   "new <host> [-- command…]",
		Short: "Start a session on a host, and attach to it",
		Example: "  " + brand.Name + " new my-vps --agent claude --dir ~/src/app\n" +
			"  " + brand.Name + " new my-vps --detach -- htop",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := client.FindHost(args[0])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			conn, err := client.Dial(ctx, h, true)
			if err != nil {
				cancel()
				return err
			}
			cols, rows := termSize()
			a := client.TerminalArgs(client.NewArgs{Agent: agent, Cwd: dir, Name: name}, cols, rows)
			if dash := cmd.ArgsLenAtDash(); dash >= 0 {
				a.Argv = args[dash:]
			}
			s, err := conn.NewSession(ctx, a)
			conn.Close()
			cancel()
			if err != nil {
				return err
			}
			if detach {
				fmt.Printf("Started %s/%s (%s, %s).\n", h.Name, s.ID, s.Name, s.Agent)
				return nil
			}
			return attach(cmd.Context(), h, s.ID, s.Name)
		},
	}
	cmd.Flags().StringVarP(&agent, "agent", "a", "shell", "the agent: claude, codex, opencode, gemini, amp, cursor, aider, goose or shell")
	cmd.Flags().StringVarP(&dir, "dir", "d", "~", "the folder on the host")
	cmd.Flags().StringVarP(&name, "name", "n", "", "a name for the session (default: the folder name)")
	cmd.Flags().BoolVar(&detach, "detach", false, "start it without attaching")
	return cmd
}

// attach runs a session in this terminal.
func attach(ctx context.Context, h client.Host, id, name string) error {
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return &client.UserError{Msg: "attach needs a terminal", Hint: "Run it from a terminal, not a pipe."}
	}
	res, err := client.Attach(ctx, client.AttachOptions{
		Host: h, Session: id, Settings: client.LoadSettings(), Interactive: true,
		Title: fmt.Sprintf("%s · %s/%s", brand.DisplayName, h.Name, name),
	})
	if err != nil {
		return err
	}
	switch {
	case res.Ended != nil && res.Ended.ExitCode == -1:
		fmt.Printf("\r\nThe session %s on %s stopped.\n", name, h.Name)
	case res.Ended != nil:
		why := fmt.Sprintf("exit code %d", res.Ended.ExitCode)
		if res.Ended.OOM {
			why = "out of memory"
		} else if res.Ended.Signal != "" {
			why = res.Ended.Signal
		}
		fmt.Printf("\r\nThe session %s on %s ended (%s).\n", name, h.Name, why)
	default:
		fmt.Printf("\r\nDetached. %s keeps running on %s.\n", name, h.Name)
	}
	return nil
}

// resolve finds a session by ID or by unique name.
func resolve(ctx context.Context, conn *client.Conn, ref string) (*client.Session, error) {
	list, err := conn.Sessions(ctx)
	if err != nil {
		return nil, err
	}
	var byName []client.Session
	for _, s := range list {
		if s.ID == ref {
			return &s, nil
		}
		if s.Name == ref {
			byName = append(byName, s)
		}
	}
	switch len(byName) {
	case 1:
		return &byName[0], nil
	case 0:
		return nil, &client.UserError{Msg: fmt.Sprintf("no session %q on %s", ref, conn.Host.Name), Hint: brand.Name + " ls --host " + conn.Host.Name}
	default:
		return nil, &client.UserError{Msg: fmt.Sprintf("%d sessions are named %q on %s", len(byName), ref, conn.Host.Name), Hint: "Use the session ID from `" + brand.Name + " ls`."}
	}
}

func withSession(ctx context.Context, target string, fn func(context.Context, client.Host, *client.Conn, *client.Session) error) error {
	hn, ref, err := client.SplitTarget(target)
	if err != nil {
		return err
	}
	h, err := client.FindHost(hn)
	if err != nil {
		return err
	}
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	conn, err := client.Dial(cctx, h, true)
	if err != nil {
		return err
	}
	defer conn.Close()
	s, err := resolve(cctx, conn, ref)
	if err != nil {
		return err
	}
	return fn(cctx, h, conn, s)
}

func attachCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "attach <host>/<session>",
		Aliases: []string{"a", "flip"},
		Short:   "Attach to a session (detach with Ctrl+\\)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var h client.Host
			var s *client.Session
			err := withSession(cmd.Context(), args[0], func(_ context.Context, hh client.Host, _ *client.Conn, ss *client.Session) error {
				h, s = hh, ss
				return nil
			})
			if err != nil {
				return err
			}
			if !s.Alive {
				return &client.UserError{Msg: fmt.Sprintf("%s on %s is %s", s.Name, h.Name, s.State), Hint: fmt.Sprintf("%s kill %s/%s, then start a new one", brand.Name, h.Name, s.ID)}
			}
			return attach(cmd.Context(), h, s.ID, s.Name)
		},
	}
}

func killCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "kill <host>/<session>",
		Aliases: []string{"rm", "delete"},
		Short:   "Stop a session and remove it",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSession(cmd.Context(), args[0], func(ctx context.Context, h client.Host, conn *client.Conn, s *client.Session) error {
				if err := conn.DeleteSession(ctx, s.ID); err != nil {
					return err
				}
				fmt.Printf("Deleted %s (%s) on %s.\n", s.Name, s.ID, h.Name)
				return nil
			})
		},
	}
}

func renameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <host>/<session> <new-name>",
		Short: "Rename a session",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSession(cmd.Context(), args[0], func(ctx context.Context, h client.Host, conn *client.Conn, s *client.Session) error {
				if err := conn.RenameSession(ctx, s.ID, args[1]); err != nil {
					return err
				}
				fmt.Printf("Renamed %s to %s on %s.\n", s.Name, args[1], h.Name)
				return nil
			})
		},
	}
}

type hostSessions struct {
	Host     string           `json:"host"`
	Sessions []client.Session `json:"sessions"`
	Error    string           `json:"error,omitempty"`
}

func lsCmd() *cobra.Command {
	var hostName string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List sessions on every host",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			hosts, err := client.LoadHosts()
			if err != nil {
				return err
			}
			if hostName != "" {
				h, err := client.FindHost(hostName)
				if err != nil {
					return err
				}
				hosts = []client.Host{h}
			}
			if len(hosts) == 0 {
				return &client.UserError{Msg: "no hosts yet", Hint: brand.Name + " host add <name>"}
			}
			results := make([]hostSessions, len(hosts))
			var wg sync.WaitGroup
			for i, h := range hosts {
				wg.Add(1)
				go func() {
					defer wg.Done()
					results[i].Host = h.Name
					ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
					defer cancel()
					conn, err := client.Dial(ctx, h, false)
					if err != nil {
						results[i].Error = err.Error()
						return
					}
					defer conn.Close()
					list, err := conn.Sessions(ctx)
					if err != nil {
						results[i].Error = err.Error()
					}
					results[i].Sessions = list
				}()
			}
			wg.Wait()
			if asJSON {
				return printJSON(results)
			}
			sort.Slice(results, func(i, j int) bool { return results[i].Host < results[j].Host })
			for _, r := range results {
				if r.Error != "" {
					fmt.Printf("%s  ◌ %s\n", r.Host, r.Error)
					continue
				}
				if len(r.Sessions) == 0 {
					fmt.Printf("%s  no sessions\n", r.Host)
					continue
				}
				for _, s := range r.Sessions {
					fmt.Printf("%-14s %-10s %-20s %-9s %-12s %6s  %s\n", r.Host, s.ID, trunc(s.Name, 20), s.Agent, stateLabel(s.State), age(s.CreatedAt), s.Cwd)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&hostName, "host", "", "only this host")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

func stateLabel(s string) string { return strings.ReplaceAll(s, "_", " ") }

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func age(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func sendCmd() *cobra.Command {
	var noEnter bool
	cmd := &cobra.Command{
		Use:     "send <host>/<session> <text>",
		Short:   "Type text into a session without attaching, then press Enter",
		Example: "  " + brand.Name + " send my-vps/auth-fix \"run the tests and fix what fails\"",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSession(cmd.Context(), args[0], func(ctx context.Context, h client.Host, conn *client.Conn, s *client.Session) error {
				if err := conn.SendText(ctx, s.ID, args[1], !noEnter); err != nil {
					return err
				}
				fmt.Printf("Sent to %s on %s.\n", s.Name, h.Name)
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&noEnter, "no-enter", false, "do not press Enter after the text")
	return cmd
}
