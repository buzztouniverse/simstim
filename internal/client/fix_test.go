package client

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalFixes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the client shares no SSH connections on Windows, so it has no control sockets to fix")
	}
	cfg, cache := t.TempDir(), shortDir(t)
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_CACHE_HOME", cache)
	if err := SaveHosts([]Host{{Name: "kept", SSH: "kept"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := writeWrapper(Host{Name: "kept", SSH: "kept"}); err != nil {
		t.Fatal(err)
	}
	if _, err := writeWrapper(Host{Name: "gone", SSH: "gone"}); err != nil {
		t.Fatal(err)
	}
	cm := filepath.Join(CacheDir(), "cm")
	_ = os.MkdirAll(cm, 0o700)
	live, err := net.Listen("unix", filepath.Join(cm, "live"))
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	dead, err := net.Listen("unix", filepath.Join(cm, "dead"))
	if err != nil {
		t.Fatal(err)
	}
	dead.(*net.UnixListener).SetUnlinkOnClose(false)
	dead.Close()

	fixes := LocalFixes(false, nil)
	if len(fixes) != 2 || fixes[0].ID != "ssh-configs" || fixes[1].ID != "control-sockets" {
		t.Fatalf("fixes %+v", fixes)
	}
	if !strings.HasSuffix(fixes[0].Command, "config-gone") || !strings.HasSuffix(fixes[1].Command, "/dead") {
		t.Fatalf("commands %q %q", fixes[0].Command, fixes[1].Command)
	}
	for _, f := range LocalFixes(true, []string{"ssh-configs", "control-sockets"}) {
		if !f.Done {
			t.Errorf("%s: %+v", f.ID, f)
		}
	}
	for p, want := range map[string]bool{
		wrapperPath(Host{Name: "kept"}): true, wrapperPath(Host{Name: "gone"}): false,
		filepath.Join(cm, "live"): true, filepath.Join(cm, "dead"): false,
	} {
		if _, err := os.Stat(p); (err == nil) != want {
			t.Errorf("%s kept = %v, want %v", p, err == nil, want)
		}
	}
	if left := LocalFixes(false, nil); len(left) != 0 {
		t.Errorf("left to fix: %+v", left)
	}
}

func TestLocalChecks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_ = os.MkdirAll(ConfigDir(), 0o700)
	conf := "[ui]\nmotoin = \"off\"\n\n[secrets]\nA_KEY = \"no-such-program-here read x\"\nB_KEY = \"sh -c 'echo b'\"\n"
	if err := os.WriteFile(settingsPath(), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	got := map[string]Check{}
	for _, c := range LocalChecks() {
		got[c.Name] = c
	}
	if c := got["config.toml"]; c.OK || !strings.Contains(c.Detail, "ui.motoin") {
		t.Errorf("config.toml: %+v", c)
	}
	if c := got["hosts.toml"]; !c.OK {
		t.Errorf("hosts.toml: %+v", c)
	}
	if c := got["secret A_KEY"]; c.OK || !strings.Contains(c.Detail, "no-such-program-here is not on PATH") {
		t.Errorf("A_KEY: %+v", c)
	}
	if c := got["secret B_KEY"]; !c.OK {
		t.Errorf("B_KEY: %+v", c)
	}
}

// shortDir is a folder with a path short enough for sockets on macOS.
func shortDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "fx")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}
