package term

import (
	"testing"

	"github.com/iamlakshya/simstim/internal/term/termtest"
)

// The client side of the attach pump: each output frame goes through the
// catch-up (with its query sanitiser while history replays) and the mode
// tracker.

const benchChunk = 32 << 10

func BenchmarkTrackerFeed(b *testing.B) {
	data := termtest.AgentOutput(benchChunk)
	t := NewTracker()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		t.Feed(data)
	}
}

func BenchmarkCatchupLive(b *testing.B) {
	data := termtest.AgentOutput(benchChunk)
	var c Catchup
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	off := int64(0)
	for b.Loop() {
		_, _ = c.Feed(off, data)
		off += int64(len(data))
	}
}

func BenchmarkCatchupHistory(b *testing.B) {
	data := termtest.AgentOutput(benchChunk)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		var c Catchup
		c.History(1 << 40) // everything is replayed history, so it is sanitised
		_, _ = c.Feed(0, data)
	}
}
