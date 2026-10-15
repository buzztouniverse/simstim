package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/tui"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Show the configuration"}
	var asJSON bool
	keymap := &cobra.Command{
		Use:   "keymap",
		Short: "Print every home-screen command and its keys (rebind them under [keymap] in config.toml)",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			entries := tui.Keymap(client.LoadSettings())
			if asJSON {
				return printJSON(entries)
			}
			for _, e := range entries {
				keys := strings.Join(e.Keys, ", ")
				if keys == "" {
					keys = "(palette only)"
				}
				fmt.Printf("%-32s %s\n", e.Command, keys)
			}
			return nil
		},
	}
	keymap.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	cmd.AddCommand(keymap, &cobra.Command{
		Use:   "path",
		Short: "Print where the configuration lives",
		Args:  cobra.NoArgs,
		Run: func(*cobra.Command, []string) {
			fmt.Println(filepath.Join(client.ConfigDir(), "config.toml"))
			fmt.Println(filepath.Join(client.ConfigDir(), "hosts.toml"))
		},
	})
	return cmd
}
