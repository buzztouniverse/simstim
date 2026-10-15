// Command cli is the client: the home screen and every command.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/version"
)

// Exit codes.
const (
	exitUser    = 1
	exitHost    = 2
	exitMissing = 3
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	root := rootCmd()
	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(printError(err))
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           brand.Name,
		Short:         "Run any coding agent on any host, and use it as if it ran here",
		Long:          brand.DisplayName + " runs coding agents in sessions on hosts you reach with SSH.\nSessions keep running when you leave. Run it with no arguments for the home screen.",
		Version:       fmt.Sprintf("%s (protocol %s)", version.Version, version.Protocol),
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHome(cmd.Context())
		},
	}
	root.AddCommand(hostCmd(), newCmd(), lsCmd(), attachCmd(), sendCmd(), killCmd(), renameCmd(), doctorCmd(), versionCmd())
	return root
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		Run: func(*cobra.Command, []string) {
			fmt.Printf("%s %s (protocol %s, host helper %s)\n", brand.Name, version.Version, version.Protocol, client.HelperVersion())
		},
	}
}

// printError prints an error with its suggested next step, and returns the exit code.
func printError(err error) int {
	var ue *client.UserError
	red, blue, bold, reset := "\x1b[31m", "\x1b[34m", "\x1b[1m", "\x1b[0m"
	if os.Getenv("NO_COLOR") != "" || !isTerminal(os.Stderr) {
		red, blue, bold, reset = "", "", "", ""
	}
	if errors.As(err, &ue) {
		fmt.Fprintf(os.Stderr, "%s%sError:%s %s\n", bold, red, reset, ue.Msg)
		if ue.Hint != "" {
			fmt.Fprintf(os.Stderr, "\n%sTry:%s %s\n", blue, reset, ue.Hint)
		}
		if ue.Hint != "" && containsAny(ue.Msg, "cannot reach", "connection") {
			return exitHost
		}
		if containsAny(ue.Msg, "no session", "no host") {
			return exitMissing
		}
		return exitUser
	}
	if errors.Is(err, client.ErrClosed) {
		fmt.Fprintf(os.Stderr, "%s%sError:%s the connection to the host closed\n\n%sTry:%s run the command again\n", bold, red, reset, blue, reset)
		return exitHost
	}
	fmt.Fprintf(os.Stderr, "%s%sError:%s %v\n", bold, red, reset, err)
	return exitUser
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if len(x) > 0 && len(s) >= len(x) {
			for i := 0; i+len(x) <= len(s); i++ {
				if s[i:i+len(x)] == x {
					return true
				}
			}
		}
	}
	return false
}
