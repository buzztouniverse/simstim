package main

import (
	"fmt"
	"strings"

	"github.com/iAmLakshya/simstim/internal/agents"
	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
	"github.com/iAmLakshya/simstim/internal/tui"
)

// keymapPage lists the launcher's commands and their default keys, from the
// command registry.
func keymapPage() []byte {
	var b strings.Builder
	b.WriteString(header("Keys", "3"))
	fmt.Fprintf(&b, "The launcher's commands and their default keys. Every command is also in the command palette (`ctrl+o`). To change a key, add the command's name under `[keymap]` in `config.toml`; `%s config keymap` prints the keys in effect.\n\n", brand.Name)
	b.WriteString("```toml\n[keymap]\n\"session: delete\" = \"x\"          # one key\n\"session: rename\" = [\"r\", \"f2\"]  # several\n\"host: details\" = \"\"             # palette only\n```\n\n")
	b.WriteString("| Command | Keys |\n| --- | --- |\n")
	for _, e := range tui.Keymap(client.Settings{}) {
		keys := "palette only"
		if len(e.Keys) > 0 {
			ks := make([]string, len(e.Keys))
			for i, k := range e.Keys {
				ks[i] = "`" + strings.ReplaceAll(k, "|", "\\|") + "`"
			}
			keys = strings.Join(ks, ", ")
		}
		fmt.Fprintf(&b, "| %s | %s |\n", e.Command, keys)
	}
	d := client.DefaultSettings()
	fmt.Fprintf(&b, "\nWhile you are attached to a session, only two keys are the client's: `%s` detaches, and `%s` pastes an image from the clipboard. Both are settings in [the configuration](config.md); everything else goes to the agent unchanged.\n", d.Attach.DetachKey, d.Attach.ImagePasteKey)
	return []byte(b.String())
}

// agentsPage lists the agent table.
func agentsPage() []byte {
	var b strings.Builder
	b.WriteString(header("Agents", "4"))
	fmt.Fprintf(&b, "Any terminal program runs in a session (`%s new <host> -- <command>`). The agents below are rows in the agent table (`internal/agents`), so %s knows how to start them, how to resume them after a host restart, and how to tell when they need you.\n\n", brand.Name, brand.DisplayName)
	b.WriteString("| Support | Meaning |\n| --- | --- |\n")
	b.WriteString("| first-class | Tested end to end on every release; status from the agent's own hooks; resumes after a reboot |\n")
	b.WriteString("| supported | Tested when the test host has it |\n")
	b.WriteString("| best-effort | In the table; contributors keep it working |\n\n")
	b.WriteString("| Agent | `--agent` | Support | Program | Starts with | Resumes with | Status from |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	rules, _ := agents.LoadRules()
	for _, a := range agents.Table {
		var from []string
		if len(a.Hooks) > 0 {
			from = append(from, "hooks")
		}
		if len(rules[a.Kind]) > 0 {
			from = append(from, "screen rules")
		}
		status := "output"
		if len(from) > 0 {
			status = strings.Join(from, ", ")
		}
		resume := "—"
		if len(a.Resume) > 0 {
			resume = "`" + strings.Join(a.Resume, " ") + "`"
		} else if len(a.Continue) > 0 {
			resume = "`" + strings.Join(a.Continue, " ") + "`"
		}
		fmt.Fprintf(&b, "| %s | `%s` | %s | `%s` | `%s` | %s | %s |\n", a.Title, a.Kind, a.Support, strings.Join(a.Binaries, "`, `"), strings.Join(a.Launch, " "), resume, status)
	}
	b.WriteString("\nTo add an agent, add a row to the table and, if it has them, screen rules in `internal/agents/rules/`. See [the contributing guide](../../CONTRIBUTING.md).\n")
	return []byte(b.String())
}

// indexPage is the reference section's home.
func indexPage() []byte {
	var b strings.Builder
	b.WriteString(header("Reference", ""))
	b.WriteString("Generated from the code, so it always matches the build.\n\n")
	b.WriteString("- [Commands](cli/index.md): every command and flag.\n")
	b.WriteString("- [Configuration](config.md): `config.toml` and `hosts.toml`.\n")
	b.WriteString("- [Keys](keymap.md): the launcher's commands and keys.\n")
	b.WriteString("- [Agents](agents.md): the agent table.\n")
	return []byte(b.String())
}
