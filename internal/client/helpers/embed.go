// Package helpers embeds the host helper binaries that the client installs on hosts.
//
// The binaries are build output and are not in git. `just helpers` (or the release
// build) writes them here before the client is built. A client built without them
// can still talk to hosts that already have a matching helper.
package helpers

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"sort"
)

//go:embed *
var files embed.FS

// For returns the helper binary for a host platform, such as "linux", "amd64".
func For(goos, goarch string) ([]byte, bool) {
	b, err := fs.ReadFile(files, "host-"+goos+"-"+goarch)
	return b, err == nil && len(b) > 0
}

// VT returns the shadow-terminal binary for a host platform, if this build has it.
func VT(goos, goarch string) ([]byte, bool) {
	b, err := fs.ReadFile(files, "vt-"+goos+"-"+goarch)
	return b, err == nil && len(b) > 0
}

// Version identifies this set of helpers. It changes whenever any helper changes,
// so each build installs into its own folder on the host and running sessions
// keep their own binary.
func Version() string {
	names, _ := fs.Glob(files, "host-*")
	vts, _ := fs.Glob(files, "vt-*")
	names = append(names, vts...)
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		b, _ := fs.ReadFile(files, n)
		h.Write([]byte(n))
		h.Write(b)
	}
	if len(names) == 0 {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// Platforms lists the embedded host platforms.
func Platforms() []string {
	names, _ := fs.Glob(files, "host-*")
	return names
}
