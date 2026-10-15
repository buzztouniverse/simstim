package e2b

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/iamlakshya/simstim/internal/brand"
	"github.com/iamlakshya/simstim/internal/client"
	"github.com/iamlakshya/simstim/internal/client/helpers"
)

// The helper's install runs over envd as over SSH: the probe, then each
// binary piped into `cat > f; sha256sum -c; chmod; mv`, in SendInput calls.
func TestInstallOverEnvd(t *testing.T) {
	bin, ok := helpers.For("linux", "amd64")
	if !ok {
		t.Skip("this build embeds no helpers (run just helpers)")
	}
	if _, err := exec.LookPath("sha256sum"); err != nil {
		t.Skip("no sha256sum on this machine")
	}
	e, envd := envdFor(t, false)
	home := t.TempDir()
	envd.mu.Lock()
	envd.home = home
	envd.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if p, err := client.Probe(ctx, e); err != nil || p.Installed {
		t.Fatalf("probe before = %+v, %v", p, err)
	}
	if err := client.Install(ctx, e, client.Platform{OS: "linux", Arch: "amd64"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".local", "share", brand.Name, "bin")
	got, err := os.ReadFile(filepath.Join(dir, client.HelperVersion(), brand.HostBinary))
	if err != nil || !bytes.Equal(got, bin) {
		t.Fatalf("the installed helper differs from the embedded one (%d bytes, %v)", len(got), err)
	}
	if link, err := os.Readlink(filepath.Join(dir, "current")); err != nil || link != client.HelperVersion() {
		t.Fatalf("current -> %q, %v", link, err)
	}
	if p, err := client.Probe(ctx, e); err != nil || !p.Installed {
		t.Fatalf("probe after = %+v, %v", p, err)
	}
}
