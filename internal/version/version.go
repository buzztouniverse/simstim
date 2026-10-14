// Package version holds build information for both binaries.
package version

// Version is set at build time with -ldflags "-X github.com/iAmLakshya/simstim/internal/version.Version=…".
var Version = "0.0.0-dev"

// Protocol is the wire protocol version. The major number changes only on a breaking change.
const Protocol = "1.0"
