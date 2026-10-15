package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client/helpers"
	"github.com/iAmLakshya/simstim/internal/version"
)

// HelperVersion names the folder the helper is installed into on hosts. It
// changes with every build of the helpers, so running sessions keep their own
// binary while new sessions use the new one.
func HelperVersion() string {
	h := helpers.Version()
	if h == "" {
		return version.Version
	}
	v := strings.NewReplacer("/", "-", "+", "-").Replace(version.Version)
	return v + "-" + h[:8]
}

// Platform is a host's operating system and architecture.
type Platform struct {
	OS, Arch string
	// Installed is true when the matching helper is already on the host.
	Installed bool
}

// Probe finds the host's platform and whether the helper is installed.
func Probe(ctx context.Context, s *SSH) (Platform, error) {
	const marker = "__HELPER_PRESENT__"
	cmd := fmt.Sprintf(`uname -sm; test -x %s && echo %s; true`, RemoteHelper(HelperVersion()), marker)
	out, err := s.Run(ctx, cmd, nil)
	if err != nil {
		return Platform{}, err
	}
	var p Platform
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == marker {
			p.Installed = true
			continue
		}
		f := strings.Fields(line)
		if len(f) == 2 && p.OS == "" {
			p.OS = strings.ToLower(f[0])
			switch f[1] {
			case "x86_64", "amd64":
				p.Arch = "amd64"
			case "aarch64", "arm64":
				p.Arch = "arm64"
			default:
				p.Arch = f[1]
			}
		}
	}
	if p.OS == "" {
		return p, fmt.Errorf("could not read the host's platform from %q", strings.TrimSpace(string(out)))
	}
	return p, nil
}

// Install uploads the helper for p, checks its SHA-256 on the host, and moves
// it into place. It never runs during a background reconnect.
func Install(ctx context.Context, s *SSH, p Platform) error {
	bin, ok := helpers.For(p.OS, p.Arch)
	if !ok {
		return &UserError{
			Msg:  fmt.Sprintf("this build has no %s helper for %s/%s", brand.Name, p.OS, p.Arch),
			Hint: "Linux hosts on amd64 and arm64 are supported. Build with `just build` to embed the helpers.",
		}
	}
	sum := sha256.Sum256(bin)
	hexsum := hex.EncodeToString(sum[:])
	ver := HelperVersion()
	dir := fmt.Sprintf(`"$HOME/.local/share/%s/bin/%s"`, brand.Name, ver)
	script := strings.Join([]string{
		"set -e",
		"umask 077",
		"d=" + dir,
		`mkdir -p "$d"`,
		`t="$d/.upload.$$"`,
		`cat > "$t"`,
		fmt.Sprintf(`echo "%s  $t" | sha256sum -c --quiet - || { rm -f "$t"; echo "checksum mismatch" >&2; exit 3; }`, hexsum),
		`chmod 755 "$t"`,
		fmt.Sprintf(`mv -f "$t" "$d/%s"`, brand.HostBinary),
		fmt.Sprintf(`ln -sfn %q "$HOME/.local/share/%s/bin/current"`, ver, brand.Name),
		"echo __INSTALLED__",
	}, "; ")
	out, err := s.Run(ctx, script, bytes.NewReader(bin))
	if err != nil {
		return err
	}
	if !strings.Contains(string(out), "__INSTALLED__") {
		return fmt.Errorf("the install did not finish: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
