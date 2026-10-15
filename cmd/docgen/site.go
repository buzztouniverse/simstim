package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iamlakshya/simstim/internal/agents"
	"github.com/iamlakshya/simstim/internal/brand"
)

// sectionWeight orders the docs site's sidebar: what users need first.
var sectionWeight = map[string]int{
	"README.md": 1, "tutorials": 2, "how-to": 3, "reference": 4, "explanation": 5,
	"roadmap.md": 6, "project": 7, "contributing": 8, "research": 9,
}

// hosted lists the Markdown files outside docs/ that the site hosts too, so
// a link to one stays on the site. The community files at the root of the
// repository make up the Project section.
var hosted = []struct {
	src, dst string
	weight   int
}{
	{"CHANGELOG.md", "project/changelog.md", 1},
	{"SUPPORT.md", "project/support.md", 2},
	{"SECURITY.md", "project/security.md", 3},
	{"CONTRIBUTING.md", "project/contributing.md", 4},
	{"CODE_OF_CONDUCT.md", "project/code-of-conduct.md", 5},
	{"GOVERNANCE.md", "project/governance.md", 6},
	{"MAINTAINERS.md", "project/maintainers.md", 7},
	{"examples/providers/README.md", "reference/provider-scripts.md", 9},
}

// projectIndex is the Project section's page. Its links are relative to
// docs/project/, as if it were a file there.
const projectIndex = `The project's own files, from the root of the repository.

| Page | What it covers |
| --- | --- |
| [Changelog](../../CHANGELOG.md) | What changed in each release, and what is not released yet |
| [Getting help](../../SUPPORT.md) | Where to ask a question, report a bug or ask for a provider |
| [Security policy](../../SECURITY.md) | How to report a security problem in private |
| [Contributing](../../CONTRIBUTING.md) | How to set up, the rules for the code, and how to send a change |
| [Code of conduct](../../CODE_OF_CONDUCT.md) | How we treat each other |
| [Governance](../../GOVERNANCE.md) | Who decides what, and how |
| [Maintainers](../../MAINTAINERS.md) | Who maintains the project |
`

// sitePages turns docs/ into the docs site's content folder. The files in
// docs/ stay plain Markdown that reads well on GitHub; here each README.md
// becomes its section's _index.md, and a page's first "# " heading becomes
// its title in front matter. docs/README.md becomes the docs' overview at
// /docs/, as the landing page has the root. Images go to the site's static
// folder, and private notes are left out.
//
// It also writes the site's data: pages.json maps each hosted file's path in
// the repository to its page, so a link between two Markdown files resolves
// to the page on the site (site/layouts/_markup/render-link.html), and
// scripts/check-site-links.py fails if a built page links to GitHub for a
// page the site hosts. brand.json and agents.json feed the landing page.
func sitePages(docs, out string) error {
	if err := os.RemoveAll(out); err != nil {
		return fmt.Errorf("clear %s: %w", out, err)
	}
	pages := map[string]string{"README.md": "/"}
	err := filepath.WalkDir(docs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(docs, p)
		if d.IsDir() {
			if rel == filepath.Join("research", "private") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(rel, "images"+string(filepath.Separator)) {
			return copyFile(p, filepath.Join(out, "static", rel))
		}
		if !strings.HasSuffix(p, ".md") || rel == filepath.Join("contributing", "decisions", "template.md") {
			return nil
		}
		dst := rel
		switch {
		case rel == "README.md":
			dst = "docs.md"
		case filepath.Base(rel) == "README.md":
			dst = filepath.Join(filepath.Dir(rel), "_index.md")
		}
		extra := ""
		if rel == "README.md" {
			extra = "linkTitle: \"Overview\"\n"
		}
		return writePage(out, pages, filepath.ToSlash(filepath.Join(docs, rel)), dst, weightOf(rel), extra)
	})
	if err != nil {
		return fmt.Errorf("write the site's pages from %s: %w", docs, err)
	}
	for _, h := range hosted {
		if err := writePage(out, pages, h.src, h.dst, h.weight, ""); err != nil {
			return err
		}
	}
	index := "---\ntitle: \"Project\"\nweight: " + strconv.Itoa(sectionWeight["project"]) + "\n---\n" + projectIndex
	if err := writeFile(filepath.Join(out, "content", "project", "_index.md"), []byte(index)); err != nil {
		return err
	}
	pages["project"] = "/project"
	landing := "---\ntitle: " + quote(brand.DisplayName) +
		"\ntype: docs\nlayout: landing\nexcludeSearch: true\ncascade:\n  - type: docs\n---\n"
	if err := writeFile(filepath.Join(out, "content", "_index.md"), []byte(landing)); err != nil {
		return err
	}
	return siteData(out, pages)
}

// writePage writes one Markdown file of the repository as a page of the
// site, with its path in the repository in front matter, and records it in
// pages.
func writePage(out string, pages map[string]string, src, dst string, weight int, extra string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	extra = "srcPath: " + quote(src) + "\n" + extra
	if err := writeFile(filepath.Join(out, "content", dst), []byte(withFrontMatter(string(b), weight, extra))); err != nil {
		return err
	}
	page := "/" + strings.TrimSuffix(filepath.ToSlash(dst), ".md")
	if path.Base(page) == "_index" {
		page = path.Dir(page)
	}
	pages[src] = page
	if base := path.Base(src); base == "README.md" || base == "_index.md" {
		pages[path.Dir(src)] = page // a link to the folder
	}
	return nil
}

// siteData writes the site's data files.
func siteData(out string, pages map[string]string) error {
	type agent struct {
		Title   string `json:"title"`
		Kind    string `json:"kind"`
		Support string `json:"support"`
		Program string `json:"program"`
	}
	list := make([]agent, 0, len(agents.Table))
	for _, a := range agents.Table {
		list = append(list, agent{Title: a.Title, Kind: a.Kind, Support: string(a.Support), Program: a.Binaries[0]})
	}
	files := map[string]any{
		"pages":  pages,
		"agents": list,
		"brand": map[string]string{
			"name": brand.Name, "displayName": brand.DisplayName, "tagline": brand.Tagline,
			"hostBinary": brand.HostBinary, "envPrefix": brand.EnvPrefix,
		},
	}
	for name, v := range files {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("encode %s: %w", name, err)
		}
		if err := writeFile(filepath.Join(out, "data", name+".json"), append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// weightOf is a page's place in the sidebar: top-level sections by
// sectionWeight, the rest by name.
func weightOf(rel string) int {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 2 && (parts[1] == "README.md" || parts[1] == "_index.md") {
		return sectionWeight[parts[0]]
	}
	if len(parts) == 1 {
		return sectionWeight[parts[0]]
	}
	return 0
}

// withFrontMatter adds front matter with the page's title, taken from its
// first "# " heading, which it removes, since the theme prints the title.
// A page that has front matter already keeps it. extra is more front matter,
// one "key: value" per line.
func withFrontMatter(md string, weight int, extra string) string {
	var keep string // front matter the page has already, without its fences
	if strings.HasPrefix(md, "---\n") {
		if end := strings.Index(md[4:], "\n---\n"); end >= 0 {
			keep, md = md[4:4+end+1], md[4+end+5:]
			if strings.Contains("\n"+keep, "\ntitle:") {
				if weight > 0 && !strings.Contains("\n"+keep, "\nweight:") {
					extra += fmt.Sprintf("weight: %d\n", weight)
				}
				return "---\n" + keep + extra + "---\n" + md
			}
		}
	}
	var title string
	var out strings.Builder
	sc := bufio.NewScanner(strings.NewReader(md))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	inFence := false
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		}
		if title == "" && !inFence && strings.HasPrefix(line, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			continue
		}
		out.WriteString(line + "\n")
	}
	fm := "---\ntitle: " + quote(title) + "\n" + keep + extra
	if weight > 0 {
		fm += fmt.Sprintf("weight: %d\n", weight)
	}
	return fm + "---\n" + strings.TrimLeft(out.String(), "\n")
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	return writeFile(dst, b)
}

func writeFile(dst string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("make %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil { //nolint:gosec // G703: dst is under the output folder named on the command line
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}
