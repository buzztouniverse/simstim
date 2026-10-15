package api

import (
	"fmt"
	"io"
)

// Secret is a secret value on its way from the laptop to one session's
// environment. Printing it with any fmt verb shows Redacted; only JSON
// carries the value, for the trip over the SSH channel.
type Secret string

// Redacted is what a Secret prints as.
const Redacted = "[redacted]"

// String hides the value.
func (Secret) String() string { return Redacted }

// GoString hides the value from %#v.
func (Secret) GoString() string { return Redacted }

// Format hides the value from every fmt verb.
func (Secret) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, Redacted) }
