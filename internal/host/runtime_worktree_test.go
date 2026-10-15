package host_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/host/hosttest"
	"github.com/iAmLakshya/simstim/internal/proto"
)

// repo makes a git repository with one commit under the home folder.
func repo(t *testing.T, h *hosttest.Host) string {
	t.Helper()
	dir := filepath.Join(h.Home, "src", "app")
	writeFile(t, filepath.Join(dir, "README"), "app\n")
	writeFile(t, filepath.Join(dir, ".gitignore"), ".env\n")
	writeFile(t, filepath.Join(dir, ".env"), "SECRET=not-in-git\n")
	writeFile(t, filepath.Join(dir, ".worktreeinclude"), "# files git ignores, for each worktree\n\n.env\n../outside\n")
	git(t, dir, "init", "-q")
	git(t, dir, "add", "README", ".gitignore", ".worktreeinclude")
	git(t, dir, "commit", "-q", "-m", "first")
	return dir
}

func TestWorktreeSessions(t *testing.T) {
	h := hosttest.New(t)
	dir := repo(t, h)
	s, err := host.New(host.NewArgs{Name: "Fix the auth bug!", Cwd: "~/src/app", Worktree: true, Argv: []string{"sleep", "60"}})
	if err != nil {
		t.Fatal(err)
	}
	g := manifest(t, s.ID).Git
	if g == nil || !g.CreatedBy || g.RepoRoot != dir || g.BaseRef != "HEAD" || s.Cwd != g.Worktree {
		t.Fatalf("git info = %+v, cwd %s", g, s.Cwd)
	}
	if want := brand.Name + "/fix-the-auth-bug-" + s.ID; g.Branch != want {
		t.Errorf("branch = %q, want %q", g.Branch, want)
	}
	if b, err := os.ReadFile(filepath.Join(g.Worktree, ".env")); err != nil || string(b) != "SECRET=not-in-git\n" {
		t.Errorf(".worktreeinclude did not copy .env: %q, %v", b, err)
	}
	if !strings.Contains(git(t, dir, "worktree", "list", "--porcelain"), "locked") {
		t.Error("the worktree is not locked while the session lives")
	}
	// Recent folders count a worktree as its repository.
	if f := host.Folders("", 0); !reflect.DeepEqual(f.Recent, []string{"~/src/app"}) {
		t.Errorf("recent = %q", f.Recent)
	}

	res, err := host.Delete(s.ID)
	if err != nil || res.Worktree != "removed the clean worktree and its branch" {
		t.Fatalf("delete = %+v, %v", res, err)
	}
	if _, err := os.Stat(g.Worktree); !os.IsNotExist(err) {
		t.Error("the clean worktree is still there")
	}
	if strings.Contains(git(t, dir, "branch"), g.Branch) {
		t.Error("the branch of the clean worktree is still there")
	}

	// A worktree with work in it is kept, and the result says what is in it.
	s, err = host.New(host.NewArgs{Name: "keep", Cwd: dir, Worktree: true, Base: "main", Argv: []string{"sleep", "60"}})
	if err != nil {
		t.Fatal(err)
	}
	g = manifest(t, s.ID).Git
	writeFile(t, filepath.Join(g.Worktree, "new.txt"), "work\n")
	git(t, g.Worktree, "add", "new.txt")
	git(t, g.Worktree, "commit", "-q", "-m", "work")
	writeFile(t, filepath.Join(g.Worktree, "README"), "changed\n")
	res, err = host.Delete(s.ID)
	if err != nil || !strings.Contains(res.Worktree, "kept the worktree") || !strings.Contains(res.Worktree, "1 new commit and 1 changed file") {
		t.Fatalf("delete = %+v, %v", res, err)
	}
	if _, err := os.Stat(g.Worktree); err != nil {
		t.Error("the worktree with work in it was removed")
	}
}

func TestWorktreeNeedsARepositoryAndABase(t *testing.T) {
	h := hosttest.New(t)
	repo(t, h)
	for _, c := range []struct {
		args host.NewArgs
		code string
	}{
		{host.NewArgs{Cwd: "~", Worktree: true}, "not_a_repo"},
		{host.NewArgs{Cwd: "~/src/app", Worktree: true, Base: "no-such-branch"}, "bad_base"},
	} {
		c.args.Argv = []string{"true"}
		_, err := host.New(c.args)
		var pe *proto.Error
		if !errors.As(err, &pe) || pe.Code != c.code {
			t.Errorf("new %+v = %v, want %s", c.args, err, c.code)
		}
	}
}

func TestFolders(t *testing.T) {
	h := hosttest.New(t)
	repo(t, h)
	for _, d := range []string{"src/other", "src/.hidden", "node_modules/pkg/.git", "deep/a/b/c/d/.git", "notes/x"} {
		if err := os.MkdirAll(filepath.Join(h.Home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(h.Home, "src", "file.txt"), "")
	f := host.Folders("~/sr", 0)
	if !reflect.DeepEqual(f.Matches, []string{"~/src/"}) {
		t.Errorf("matches for ~/sr = %q", f.Matches)
	}
	if !reflect.DeepEqual(f.Repos, []string{"~/src/app"}) {
		t.Errorf("repos = %q: node_modules and folders too deep are skipped", f.Repos)
	}
	if f := host.Folders("~/src/", 1); len(f.Matches) != 1 {
		t.Errorf("limit 1 gave %q", f.Matches)
	}
	if f := host.Folders("~/src/", 0); !reflect.DeepEqual(f.Matches, []string{"~/src/app/", "~/src/other/"}) {
		t.Errorf("matches for ~/src/ = %q: files and hidden folders are left out", f.Matches)
	}
	if f := host.Folders("~/src/.h", 0); !reflect.DeepEqual(f.Matches, []string{"~/src/.hidden/"}) {
		t.Errorf("a hidden folder asked for by name: %q", f.Matches)
	}
	start(t, "in-notes", "sleep", "60")
	if f := host.Folders("/no/such/place/", 0); len(f.Matches) != 0 || !reflect.DeepEqual(f.Recent, []string{"~"}) {
		t.Errorf("folders = %+v", f)
	}
}
