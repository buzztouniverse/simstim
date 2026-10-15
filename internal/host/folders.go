package host

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/iamlakshya/simstim/internal/api"
	"github.com/iamlakshya/simstim/internal/paths"
)

// FoldersResult is what the new-session dialog completes folders from.
type FoldersResult = api.FoldersResult

// Folders completes a folder path on this host, and suggests the folders of
// recent sessions and the git repositories under the home folder. Paths under
// the home folder are written with ~.
func Folders(prefix string, limit int) FoldersResult {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	home, _ := os.UserHomeDir()
	tilde := func(p string) string {
		if p == home {
			return "~"
		}
		if rest, ok := strings.CutPrefix(p, home+"/"); ok {
			return "~/" + rest
		}
		return p
	}
	res := FoldersResult{Matches: []string{}, Recent: []string{}, Repos: []string{}}

	if prefix != "" {
		full := expandHome(prefix)
		dir, base := full, ""
		if !strings.HasSuffix(prefix, "/") {
			dir, base = filepath.Dir(full), filepath.Base(full)
		}
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if !strings.HasPrefix(name, base) || (strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".")) {
					continue
				}
				p := filepath.Join(dir, name)
				if st, err := os.Stat(p); err == nil && st.IsDir() {
					res.Matches = append(res.Matches, tilde(p)+"/")
				}
				if len(res.Matches) >= limit {
					break
				}
			}
		}
	}

	// Folders of recent sessions, newest first (worktrees count as their repository).
	if list, err := List(); err == nil {
		seen := map[string]bool{}
		for _, s := range list {
			d := s.Cwd
			if m, err := LoadManifest(s.ID); err == nil && m.Git != nil {
				d = m.Git.RepoRoot
			}
			if t := tilde(d); !seen[t] {
				seen[t] = true
				res.Recent = append(res.Recent, t)
			}
			if len(res.Recent) >= 8 {
				break
			}
		}
	}

	// Git repositories under home, three levels deep, within a small time budget.
	deadline := time.Now().Add(300 * time.Millisecond)
	skip := map[string]bool{"node_modules": true, "vendor": true, "target": true, "dist": true, "build": true}
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > 3 || len(res.Repos) >= limit || time.Now().After(deadline) {
			return
		}
		if dir == paths.DataDir() || dir == paths.StateDir() {
			return
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil && dir != home {
			res.Repos = append(res.Repos, tilde(dir))
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			n := e.Name()
			if !e.IsDir() || strings.HasPrefix(n, ".") || skip[n] {
				continue
			}
			walk(filepath.Join(dir, n), depth+1)
		}
	}
	walk(home, 0)
	sort.Strings(res.Repos)
	return res
}
