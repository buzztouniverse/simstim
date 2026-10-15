package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/keys"
)

func doctorCmd() *cobra.Command {
	var fix, yes bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check this machine and its terminal",
		Long: "Checks this computer: ssh, the config files and the secrets' commands. With --fix it\n" +
			"offers to fix what it can: SSH configs it generated for hosts you removed, and\n" +
			"ControlMaster sockets whose master died. Each fix is shown before it runs.\n" +
			"`host doctor <name>` checks a host.",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			fmt.Println("This computer:")
			printChecks(os.Stdout, client.LocalChecks())
			fixes := client.LocalFixes(false, nil)
			if !fix {
				for _, f := range fixes {
					fmt.Printf("  ✕ %s\n", f.Problem)
				}
				if len(fixes) > 0 {
					fmt.Printf("\nTry: %s doctor --fix\n", brand.Name)
				}
				return nil
			}
			return offerFixes(os.Stdin, os.Stdout, fixes, yes, func(ids []string) ([]client.Fix, error) {
				return client.LocalFixes(true, ids), nil
			})
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "offer to fix the problems found, one by one")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "with --fix, apply every fix without asking")
	cmd.AddCommand(&cobra.Command{
		Use:   "terminal",
		Short: "Ask this terminal what it supports (keys, colours, synchronized output…)",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return terminalProbe() },
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "keys",
		Short: "Show every key as the terminal sends it, and which keys " + brand.DisplayName + " keeps",
		Long: "Turns on the kitty keyboard protocol, then prints each key press as bytes.\n" +
			"To check the path to a host, start a session that runs the same tester there:\n" +
			"  " + brand.Name + " new <host> -- " + brand.HostBinary + " keytest",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return keyTester() },
	})
	return cmd
}

func keyTester() error {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return &client.UserError{Msg: "the key tester needs a terminal", Hint: "Run it from a terminal."}
	}
	s := client.LoadSettings()
	dk, err := keys.Parse(s.Attach.DetachKey)
	if err != nil {
		return err
	}
	pk, err := keys.Parse(s.Attach.ImagePasteKey)
	if err != nil {
		return err
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(fd, old) }()
	// Kitty keyboard flags 1|2|4|8|16: report everything, with event types.
	fmt.Print("\x1b[>31u\x1b[?2004h")
	defer fmt.Print("\x1b[<u\x1b[?2004l")
	fmt.Printf("Press keys. %s owns %s (detach) and %s (image paste). Press q twice to quit.\r\n\r\n", brand.DisplayName, dk, pk)
	f := keys.New(dk, pk)
	buf := make([]byte, 4096)
	qs := 0
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil
		}
		for _, p := range f.Feed(buf[:n]) {
			label := ""
			switch p.Kind {
			case keys.Match:
				label = []string{"  ← detach key", "  ← image paste key"}[p.Index]
				if p.Release {
					label += " (release)"
				}
			case keys.Paste:
				label = "  ← a paste"
			}
			fmt.Printf("%s%s\r\n", strconv.Quote(string(p.Raw)), label)
			if string(p.Raw) == "q" || string(p.Raw) == "\x1b[113u" {
				qs++
				if qs >= 2 {
					return nil
				}
			} else if p.Kind != keys.Match || !p.Release {
				qs = 0
			}
		}
	}
}
