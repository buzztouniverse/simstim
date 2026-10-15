// Package version holds build information for both binaries.
package version

import (
	"runtime/debug"
	"strings"
)

// Version is set at build time with -ldflags "-X github.com/iamlakshya/simstim/internal/version.Version=…".
// A build without it (go install <module>@vX.Y.Z) takes the module's version
// from Go's build information.
var Version = "0.0.0-dev"

// Protocol is the wire protocol version. The major number changes only on a breaking change.
const Protocol = "1.0"

func init() {
	if Version != "0.0.0-dev" {
		return
	}
	if v := moduleVersion(); v != "" {
		Version = v
	}
}

// moduleVersion is the main module's version when the go command recorded
// one, without its leading "v": "0.1.0" for go install …@v0.1.0.
func moduleVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	v := bi.Main.Version
	if v == "" || v == "(devel)" || strings.Contains(v, "+dirty") {
		return ""
	}
	return strings.TrimPrefix(v, "v")
}

// Module is the client's module path, such as github.com/<owner>/<repo>, from
// Go's build information; it says where the releases are.
func Module() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Path != "" {
		return bi.Main.Path
	}
	return ""
}
