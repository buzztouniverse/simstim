package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/cli"
)

// root builds the client's command tree for the docs.
func root() *cobra.Command {
	r := cli.Root()
	r.DisableAutoGenTag = true
	return r
}

// pageName is a command's page: "host add" becomes host_add.md, and the root
// command is index.md. File names leave out the product's name, so a rename
// changes no paths.
func pageName(path string) string {
	rest := strings.TrimSpace(strings.TrimPrefix(path, brand.Name))
	if rest == "" {
		return "_index.md"
	}
	return strings.ReplaceAll(rest, " ", "_") + ".md"
}

// cliPages adds one page per visible command under cli/.
func cliPages(pages map[string][]byte) error {
	var walk func(c *cobra.Command) error
	walk = func(c *cobra.Command) error {
		if !c.IsAvailableCommand() && c.HasParent() || c.IsAdditionalHelpTopicCommand() {
			return nil
		}
		var b bytes.Buffer
		link := func(name string) string {
			return pageName(strings.ReplaceAll(strings.TrimSuffix(name, ".md"), "_", " "))
		}
		if err := doc.GenMarkdownCustom(c, &b, link); err != nil {
			return fmt.Errorf("markdown for %s: %w", c.CommandPath(), err)
		}
		title := c.CommandPath()
		weight := ""
		if !c.HasParent() {
			title, weight = "Commands", "1"
		}
		pages[filepath.Join("cli", pageName(c.CommandPath()))] = []byte(header(title, weight) + tidy(b.String()))
		for _, sub := range c.Commands() {
			if err := walk(sub); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root())
}

// tidy makes cobra's Markdown pass the docs checks: it drops the top
// heading (the front matter has the title), lifts "###" sections to "##",
// marks code blocks as text, and turns tabs into spaces.
func tidy(md string) string {
	if strings.HasPrefix(md, "## ") {
		if i := strings.Index(md, "\n"); i >= 0 {
			md = md[i+1:]
		}
	}
	lines := strings.Split(md, "\n")
	inFence := false
	for i, l := range lines {
		switch {
		case strings.HasPrefix(l, "```"):
			if !inFence && l == "```" {
				lines[i] = "```text"
			}
			inFence = !inFence
		case !inFence && strings.HasPrefix(l, "### "):
			lines[i] = strings.TrimPrefix(l, "#")
		}
		lines[i] = strings.ReplaceAll(lines[i], "\t", "    ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

// manPages writes section 1 man pages, one per command.
func manPages(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("make %s: %w", dir, err)
	}
	h := &doc.GenManHeader{Title: strings.ToUpper(brand.Name), Section: "1", Source: brand.DisplayName, Manual: brand.DisplayName + " manual"}
	if err := doc.GenManTree(root(), h, dir); err != nil {
		return fmt.Errorf("man pages: %w", err)
	}
	return nil
}

// completions writes the bash, zsh, fish and PowerShell completion scripts.
func completions(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("make %s: %w", dir, err)
	}
	r := root()
	for _, c := range []struct {
		file string
		gen  func(*bytes.Buffer) error
	}{
		{brand.Name + ".bash", func(b *bytes.Buffer) error { return r.GenBashCompletionV2(b, true) }},
		{"_" + brand.Name, func(b *bytes.Buffer) error { return r.GenZshCompletion(b) }},
		{brand.Name + ".fish", func(b *bytes.Buffer) error { return r.GenFishCompletion(b, true) }},
		{brand.Name + ".ps1", func(b *bytes.Buffer) error { return r.GenPowerShellCompletionWithDesc(b) }},
	} {
		var b bytes.Buffer
		if err := c.gen(&b); err != nil {
			return fmt.Errorf("completion %s: %w", c.file, err)
		}
		if err := os.WriteFile(filepath.Join(dir, c.file), b.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write completion: %w", err)
		}
	}
	return nil
}
