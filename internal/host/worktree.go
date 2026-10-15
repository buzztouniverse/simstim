package host

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iamlakshya/simstim/internal/brand"
	"github.com/iamlakshya/simstim/internal/paths"
)

// GitInfo is the git part of a manifest, for a session that has its own worktree.
type GitInfo struct {
	RepoRoot  string `json:"repo_root"`
	Worktree  string `json:"worktree"`
	Branch    string `json:"branch"`
	BaseRef   string `json:"base_ref"`
	BaseSHA   string `json:"base_sha"`
	CreatedBy bool   `json:"created_by_us"`
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		return s, fmt.Errorf("git %s: %s", strings.Join(args, " "), s)
	}
	return s, nil
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns a name into a branch-safe slug.
func Slug(s string) string {
	s = strings.Trim(slugRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(s) > 40 {
		s = strings.TrimRight(s[:40], "-")
	}
	if s == "" {
		s = "task"
	}
	return s
}

// WorktreesDir holds the worktrees of all sessions.
func WorktreesDir() string { return filepath.Join(paths.DataDir(), "worktrees") }

// makeWorktree creates a git worktree for a new session: a new branch from base,
// in a folder of its own, locked while the session lives.
func makeWorktree(cwd, name, id, base string) (*GitInfo, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, UserError("no_git", "git is not installed on this host", "Install git on the host, or start the session without a worktree.")
	}
	root, err := git(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, UserError("not_a_repo", fmt.Sprintf("%s is not inside a git repository", cwd), "Pick a folder inside a repository, or start without a worktree.")
	}
	if base == "" {
		base = "HEAD"
	}
	sha, err := git(root, "rev-parse", "--verify", base+"^{commit}")
	if err != nil {
		return nil, UserError("bad_base", fmt.Sprintf("%s is not a commit in %s", base, root), "Pass an existing branch or commit as the base.")
	}
	slug := Slug(name)
	branch := brand.Name + "/" + slug + "-" + id
	wt := filepath.Join(WorktreesDir(), filepath.Base(root), slug+"-"+id)
	if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
		return nil, fmt.Errorf("make the folder for the worktree: %w", err)
	}
	if _, err := git(root, "worktree", "add", "-b", branch, wt, sha); err != nil {
		return nil, UserError("worktree", err.Error(), "Check the repository's state with git status.")
	}
	_, _ = git(root, "worktree", "lock", "--reason", brand.DisplayName+" session "+id, wt)
	copyWorktreeInclude(root, wt)
	return &GitInfo{RepoRoot: root, Worktree: wt, Branch: branch, BaseRef: base, BaseSHA: sha, CreatedBy: true}, nil
}

// copyWorktreeInclude copies files listed in .worktreeinclude (for example .env,
// which git ignores) from the repository into the new worktree.
func copyWorktreeInclude(root, wt string) {
	f, err := os.Open(filepath.Join(root, ".worktreeinclude"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(root, line))
		for _, src := range matches {
			rel, err := filepath.Rel(root, src)
			if err != nil || strings.HasPrefix(rel, "..") {
				continue
			}
			_ = copyFile(src, filepath.Join(wt, rel))
		}
	}
}

func copyFile(src, dst string) error {
	st, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if !st.Mode().IsRegular() {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	return nil
}

// removeWorktree removes a session's worktree and branch only when that loses
// nothing: no uncommitted changes and no commits beyond the base. It reports
// what it did, for the user.
func removeWorktree(g *GitInfo) string {
	if g == nil || !g.CreatedBy {
		return ""
	}
	if _, err := os.Stat(g.Worktree); err != nil {
		return ""
	}
	_, _ = git(g.RepoRoot, "worktree", "unlock", g.Worktree)
	dirty, _ := git(g.Worktree, "status", "--porcelain")
	ahead, _ := git(g.RepoRoot, "rev-list", "--count", g.BaseSHA+".."+g.Branch)
	if dirty != "" || (ahead != "" && ahead != "0") {
		return fmt.Sprintf("kept the worktree %s (branch %s): it has %s and %s", g.Worktree, g.Branch,
			plural(atoi(ahead), "new commit", "new commits"), plural(countLines(dirty), "changed file", "changed files"))
	}
	if _, err := git(g.RepoRoot, "worktree", "remove", g.Worktree); err != nil {
		return "could not remove the worktree: " + err.Error()
	}
	_, _ = git(g.RepoRoot, "branch", "-D", g.Branch)
	return "removed the clean worktree and its branch"
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
