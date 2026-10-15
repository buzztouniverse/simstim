package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/iamlakshya/simstim/internal/agents"
	"github.com/iamlakshya/simstim/internal/brand"
)

// hosted lists the Markdown files outside docs/ that the site hosts too, so
// a link to one stays on the site. The community files at the root of the
// repository are the Project tab's pages.
var hosted = map[string]string{
	"CHANGELOG.md":                 "project/changelog.md",
	"SUPPORT.md":                   "project/support.md",
	"SECURITY.md":                  "project/security.md",
	"CONTRIBUTING.md":              "project/contributing.md",
	"CODE_OF_CONDUCT.md":           "project/code-of-conduct.md",
	"GOVERNANCE.md":                "project/governance.md",
	"MAINTAINERS.md":               "project/maintainers.md",
	"examples/providers/README.md": "reference/provider-scripts.md",
}

// projectIndex is the Project tab's overview. Its links are relative to
// docs/project/, as if it were a file there.
const projectIndex = `# The project

How the project is run, what changed, what comes next, and how to help build it.

| Page | What it covers |
| --- | --- |
| [Changelog](../../CHANGELOG.md) | What changed in each release, and what is not released yet |
| [Roadmap](../roadmap.md) | Every candidate feature, scored by demand, impact and effort |
| [Getting help](../../SUPPORT.md) | Where to ask a question, report a bug or ask for a provider |
| [Security policy](../../SECURITY.md) | How to report a security problem in private |
| [How to contribute](../../CONTRIBUTING.md) | How to set up, the rules for the code, and how to send a change |
| [Contributor guide](../contributing/README.md) | How the project is built, tested, documented and released |
| [Decision records](../contributing/decisions/README.md) | Each design choice and its reasons |
| [Research notes](../research/README.md) | The evidence behind the design |
| [Governance](../../GOVERNANCE.md) | Who decides what, and how |
| [Maintainers](../../MAINTAINERS.md) | Who maintains the project |
| [Code of conduct](../../CODE_OF_CONDUCT.md) | How we treat each other |
`

// page is one Markdown file of the repository as a page of the site.
type page struct {
	src string // its path in the repository, or "" for a generated page
	dst string // its path under content/docs
	url string // its address on the site, such as /how-to/install/
	md  string // its Markdown, before links are rewritten
}

// sitePages turns docs/ and the community files into the docs site's
// content (decision 0022). The files stay plain Markdown that reads well on
// GitHub; here each README.md becomes its folder's index page, a page's
// first "# " heading becomes its title, and every link between two hosted
// files becomes a link between their pages, so a reader never leaves the
// site for a page it hosts. Links to other files (code, scripts) go to
// their source on GitHub. Images are copied next to the pages.
//
// It writes, under out (the site's src folder):
//   - content/docs/: the pages;
//   - generated/pages.json: every hosted file's path in the repository and
//     its page, for scripts/check-site-links.py;
//   - generated/sidebar.json: the three tabs and their sidebars (sitenav.go);
//   - generated/brand.json and agents.json, for the landing page.
func sitePages(docs, out, repo string) error {
	content := filepath.Join(out, "content", "docs")
	for _, dir := range []string{content, filepath.Join(out, "generated"), filepath.Join(out, "..", "public", "images")} {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("clear %s: %w", dir, err)
		}
	}
	pages, err := collect(docs)
	if err != nil {
		return err
	}
	byURL := map[string]string{"README.md": "/"}
	for _, p := range pages {
		byURL[p.src] = p.url
		if base := path.Base(p.src); base == "README.md" || base == "_index.md" {
			byURL[path.Dir(p.src)] = p.url // a link to the folder
		}
	}
	byURL["docs/project"] = "/project/"
	for _, p := range pages {
		md := rewriteLinks(p.md, p.src, p.url, byURL, repo)
		if err := writeFile(filepath.Join(content, p.dst), []byte(withFrontMatter(md, p, repo))); err != nil {
			return err
		}
	}
	if err := copyImages(docs, filepath.Join(out, "..", "public", "images")); err != nil {
		return err
	}
	nav, err := sidebar(docs, byURL)
	if err != nil {
		return err
	}
	return siteData(filepath.Join(out, "generated"), byURL, nav)
}

// collect reads every page the site hosts.
func collect(docs string) ([]page, error) {
	var pages []page
	err := filepath.WalkDir(docs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(docs, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "research/private" || rel == "images" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".md") || rel == "contributing/decisions/template.md" {
			return nil
		}
		dst := rel
		switch {
		case rel == "README.md":
			dst = "docs.md"
		case path.Base(rel) == "README.md" || path.Base(rel) == "_index.md":
			dst = path.Join(path.Dir(rel), "index.md")
		}
		return add(&pages, path.Join("docs", rel), dst, "")
	})
	if err != nil {
		return nil, fmt.Errorf("read the pages in %s: %w", docs, err)
	}
	srcs := make([]string, 0, len(hosted))
	for src := range hosted {
		srcs = append(srcs, src)
	}
	sort.Strings(srcs)
	for _, src := range srcs {
		if err := add(&pages, src, hosted[src], ""); err != nil {
			return nil, err
		}
	}
	return pages, add(&pages, "docs/project/README.md", "project/index.md", projectIndex)
}

// add reads one page, or takes md as its text when it is generated.
func add(pages *[]page, src, dst, md string) error {
	if md == "" {
		b, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read %s: %w", src, err)
		}
		md = string(b)
	}
	url := "/" + strings.TrimSuffix(dst, ".md") + "/"
	if path.Base(dst) == "index.md" {
		url = "/" + path.Dir(dst) + "/"
	}
	*pages = append(*pages, page{src: src, dst: dst, url: url, md: md})
	return nil
}

// withFrontMatter gives a page the front matter the site needs: its title,
// from its first "# " heading, which it removes, since the site prints the
// title; a description, from its first paragraph; where its source is; and
// where to suggest a change, unless the page is generated. Front matter the
// page has already is kept, apart from the Hugo-only weight.
func withFrontMatter(md string, p page, repo string) string {
	var keep []string
	if strings.HasPrefix(md, "---\n") {
		if end := strings.Index(md[4:], "\n---\n"); end >= 0 {
			for _, l := range strings.Split(md[4:4+end], "\n") {
				if !strings.HasPrefix(l, "weight:") {
					keep = append(keep, l)
				}
			}
			md = md[4+end+5:]
		}
	}
	title, body := splitTitle(md)
	for _, l := range keep {
		if strings.HasPrefix(l, "title:") {
			title = ""
		}
	}
	var fm strings.Builder
	fm.WriteString("---\n")
	if title != "" {
		fm.WriteString("title: " + quote(title) + "\n")
	}
	for _, l := range keep {
		fm.WriteString(l + "\n")
	}
	if d := describe(body); d != "" {
		fm.WriteString("description: " + quote(d) + "\n")
	}
	fm.WriteString("srcPath: " + quote(p.src) + "\n")
	generated := strings.HasPrefix(p.src, "docs/reference/") && p.src != "docs/reference/README.md" || p.src == "docs/project/README.md"
	if generated {
		fm.WriteString("editUrl: false\ngenerated: true\n")
	} else {
		fm.WriteString("editUrl: " + quote(repo+"/edit/main/"+p.src) + "\n")
	}
	fm.WriteString("---\n")
	return fm.String() + strings.TrimLeft(body, "\n")
}

// splitTitle takes a page's first "# " heading outside code as its title.
func splitTitle(md string) (string, string) {
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
	return title, out.String()
}

// describe is a page's first paragraph as plain text, cut to one or two
// sentences: the page's description for search engines and link previews.
func describe(body string) string {
	for _, para := range strings.Split(body, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" || strings.ContainsAny(para[:1], "#|<`>-*![0123456789") || strings.HasPrefix(para, "**") {
			continue
		}
		text := strings.Join(strings.Fields(para), " ")
		text = mdLink.ReplaceAllString(text, "$1")
		text = strings.NewReplacer("`", "", "**", "", "_", "").Replace(text)
		if len(text) > 200 {
			if i := strings.Index(text[100:], ". "); i >= 0 && i+101 < 240 {
				text = text[:i+101]
			} else {
				text = text[:197] + "…"
			}
		}
		return text
	}
	return ""
}

// copyImages copies docs/images into the site's public folder.
func copyImages(docs, dst string) error {
	files, err := filepath.Glob(filepath.Join(docs, "images", "*"))
	if err != nil {
		return fmt.Errorf("list the images: %w", err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if err := writeFile(filepath.Join(dst, filepath.Base(f)), b); err != nil {
			return err
		}
	}
	return nil
}

// siteData writes the site's data files.
func siteData(dir string, pages map[string]string, nav []tab) error {
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
		"pages":   pages,
		"agents":  list,
		"sidebar": nav,
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
		if err := writeFile(filepath.Join(dir, name+".json"), append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
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
