package helpers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The key in the code is the one in .github/release-signers and install.sh.
func TestReleaseSignerMatchesTheRepo(t *testing.T) {
	for _, f := range []string{"../../../.github/release-signers", "../../../install.sh"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		key := ReleaseSigner[strings.Index(ReleaseSigner, "ssh-ed25519"):]
		if !strings.Contains(string(b), key) {
			t.Errorf("%s does not hold the release key in the code", f)
		}
	}
}

// fakeRelease serves a signed release with a helper bundle, signed by a
// throwaway key that the test trusts instead of the release key.
func fakeRelease(t *testing.T, version string, tamper bool) *httptest.Server {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("needs ssh-keygen")
	}
	dir := t.TempDir()
	key := filepath.Join(dir, "key")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "test", "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	pub, _ := os.ReadFile(key + ".pub")
	f := strings.Fields(string(pub))
	old := signer
	signer = `release namespaces="file" ` + f[0] + " " + f[1]
	t.Cleanup(func() { signer = old })

	var bundle bytes.Buffer
	zw := gzip.NewWriter(&bundle)
	tw := tar.NewWriter(zw)
	for name, body := range map[string]string{"host-linux-amd64": "HOST", "vt-linux-amd64": "VT"} {
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = zw.Close()
	sum := sha256.Sum256(bundle.Bytes())
	sums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), BundleName(version))
	sumsFile := filepath.Join(dir, "checksums.txt")
	_ = os.WriteFile(sumsFile, []byte(sums), 0o600)
	if out, err := exec.Command("ssh-keygen", "-q", "-Y", "sign", "-n", "file", "-f", key, sumsFile).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	sig, _ := os.ReadFile(sumsFile + ".sig")
	if tamper {
		sums = strings.Replace(sums, "  ", "  x", 1)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Base(r.URL.Path) {
		case "checksums.txt":
			_, _ = w.Write([]byte(sums))
		case "checksums.txt.sig":
			_, _ = w.Write(sig)
		case BundleName(version):
			_, _ = w.Write(bundle.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// resetFetched makes this test a build without embedded helpers, with none
// fetched yet.
func resetFetched(t *testing.T) {
	t.Helper()
	saved := files
	files = embed.FS{}
	t.Cleanup(func() { files = saved })
	fetchMu.Lock()
	fetched = nil
	fetchMu.Unlock()
	t.Cleanup(func() {
		fetchMu.Lock()
		fetched = nil
		fetchMu.Unlock()
	})
}

func TestFetchVerifiesAndCaches(t *testing.T) {
	resetFetched(t)
	srv := fakeRelease(t, "9.9.9", false)
	cache := t.TempDir()
	if err := Fetch(context.Background(), "9.9.9", srv.URL, cache); err != nil {
		t.Fatal(err)
	}
	if b, ok := For("linux", "amd64"); !ok || string(b) != "HOST" {
		t.Fatalf("helper = %q, %v", b, ok)
	}
	if b, ok := VT("linux", "amd64"); !ok || string(b) != "VT" {
		t.Fatalf("vt = %q, %v", b, ok)
	}
	if Version() == "" || len(Platforms()) != 1 {
		t.Errorf("version %q, platforms %v", Version(), Platforms())
	}
	// A second client reads the cache, with no network.
	resetFetched(t)
	srv.Close()
	if err := Fetch(context.Background(), "9.9.9", srv.URL, cache); err != nil {
		t.Fatalf("from the cache: %v", err)
	}
}

func TestFetchRefusesAChangedRelease(t *testing.T) {
	resetFetched(t)
	srv := fakeRelease(t, "9.9.9", true)
	err := Fetch(context.Background(), "9.9.9", srv.URL, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("err = %v, want a signature error", err)
	}
	if _, ok := For("linux", "amd64"); ok {
		t.Error("a helper from a bad release was kept")
	}
}
