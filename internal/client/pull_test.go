package client

import (
	"reflect"
	"testing"
)

func TestParseRepoInfo(t *testing.T) {
	r, err := parseRepoInfo("/home/dev/src/app\nauth-fix\n      3\n")
	if err != nil || r != (RepoInfo{Root: "/home/dev/src/app", Branch: "auth-fix", Dirty: 3}) {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := parseRepoInfo("fatal: not a git repository\n"); err == nil {
		t.Error("garbage parsed")
	}
}

func TestFetchArgs(t *testing.T) {
	h := Host{Name: "box", SSH: "dev@box"}
	got := FetchArgs(h, "/home/dev/src/app", "auth-fix", "review/auth-fix", false)
	want := []string{"fetch", "--no-tags", "dev@box:/home/dev/src/app", "refs/heads/auth-fix:refs/heads/review/auth-fix"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := FetchArgs(h, "/r", "b", "b", true); got[3] != "+refs/heads/b:refs/heads/b" {
		t.Errorf("force: %q", got[3])
	}
}

func TestShellWord(t *testing.T) {
	for in, want := range map[string]string{"plain": "plain", "a b": "'a b'", "it's": `'it'\''s'`, "ControlPath=/x/%C": "'ControlPath=/x/%C'"} {
		if got := shellWord(in); got != want {
			t.Errorf("shellWord(%q) = %q, want %q", in, got, want)
		}
	}
}
