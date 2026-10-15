package host

import (
	"testing"

	"github.com/iamlakshya/simstim/internal/agents"
	"github.com/iamlakshya/simstim/internal/term/termtest"
)

// The holder feeds each chunk of the agent's output to its screen model,
// and reads the screen on each tick to judge the agent's state.

func BenchmarkScreenModelFeed(b *testing.B) {
	data := termtest.AgentOutput(32 << 10)
	m := newScreenModel(120, 40)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		m.feed(data)
	}
}

func BenchmarkScreenDetect(b *testing.B) {
	m := newScreenModel(120, 40)
	m.feed(termtest.AgentOutput(32 << 10))
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = agents.Detect("claude", m.screen())
	}
}
