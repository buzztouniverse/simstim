// Package brand holds the product name. Nothing else in the code base may spell it.
//
// The name may change. To rename the project, change Name here and follow
// docs/renaming.md, which also covers the module path, the build files and the docs.
package brand

import "strings"

// Name is the identifier used for binaries, folders, environment variables,
// systemd units and the wire protocol. Lowercase letters only.
const Name = "simstim"

// DisplayName is the name shown to people.
const DisplayName = "simstim"

// Tagline is the one-line description shown on the first-run screen.
const Tagline = "jack in to any host"

// HostBinary is the name of the helper installed on each host.
const HostBinary = Name + "-host"

// VTBinary is the name of the shadow-terminal helper on each host.
const VTBinary = Name + "-vt"

// UnitPrefix starts the name of each session's systemd unit.
const UnitPrefix = Name + "-s-"

// Slice is the systemd slice that holds all sessions.
const Slice = Name + ".slice"

// EnvPrefix starts every environment variable the product reads or sets.
var EnvPrefix = strings.ToUpper(Name) + "_"

// Env returns the full name of an environment variable, for example Env("SESSION").
func Env(suffix string) string { return EnvPrefix + suffix }
