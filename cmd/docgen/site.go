package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// sectionWeight orders the docs site's sidebar: what users need first.
var sectionWeight = map[string]int{
	"tutorials": 1, "how-to": 2, "reference": 3, "explanation": 4,
	"roadmap.md": 5, "contributing": 6, "research": 7,
}

// sitePages turns docs/ into the docs site's content folder. The files in
// docs/ stay plain Markdown that reads well on GitHub; here each README.md
// becomes its section's _index.md, and a page's first "# " heading becomes
// its title in front matter. Images go to the site's static folder, and
// private notes are left out.
func sitePages(docs, out string) error {
	if err := os.RemoveAll(out); err != nil {
		return fmt.Errorf("clear %s: %w", out, err)
	}
	return filepath.WalkDir(docs, func(p string, d fs.DirEntry, err error) error {
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
		if filepath.Base(rel) == "README.md" {
			dst = filepath.Join(filepath.Dir(rel), "_index.md")
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		page := withFrontMatter(string(b), weightOf(rel))
		target := filepath.Join(out, "content", dst)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("make %s: %w", filepath.Dir(target), err)
		}
		if err := os.WriteFile(target, []byte(page), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
		return nil
	})
}

// weightOf is a page's place in the sidebar: top-level sections by
// sectionWeight, the rest by name.
func weightOf(rel string) int {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 2 && parts[1] == "README.md" {
		return sectionWeight[parts[0]]
	}
	if len(parts) == 1 {
		return sectionWeight[parts[0]]
	}
	return 0
}

// withFrontMatter adds front matter with the page's title, taken from its
// first "# " heading, which it removes, since the theme prints the title.
// A page that has front matter already keeps it.
func withFrontMatter(md string, weight int) string {
	var keep string // front matter the page has already, without its fences
	if strings.HasPrefix(md, "---\n") {
		if end := strings.Index(md[4:], "\n---\n"); end >= 0 {
			keep, md = md[4:4+end+1], md[4+end+5:]
			if strings.Contains("\n"+keep, "\ntitle:") {
				return "---\n" + keep + "---\n" + md
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
	fm := "---\ntitle: " + quote(title) + "\n" + keep
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
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("make %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}
