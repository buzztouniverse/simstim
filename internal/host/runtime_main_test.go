package host_test

import (
	"testing"

	"github.com/iamlakshya/simstim/internal/host/hosttest"
)

// The tests in this binary may start it again as a session's holder.
func TestMain(m *testing.M) { hosttest.Main(m) }
