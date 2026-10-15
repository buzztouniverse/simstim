package client

import (
	"os"
	"testing"

	"github.com/iAmLakshya/simstim/internal/proto"
	"github.com/iAmLakshya/simstim/internal/term"
	"github.com/iAmLakshya/simstim/internal/term/termtest"
)

// The client end of the attach pump: an output frame from the host is
// decoded, trimmed to what the terminal lacks, written to the terminal, and
// fed to the mode tracker.
func BenchmarkAttachOutputFrames(b *testing.B) {
	out, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer out.Close()
	a := &attacher{o: AttachOptions{Out: out}, tracker: term.NewTracker()}
	data := termtest.AgentOutput(32 << 10)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	off := int64(0)
	for b.Loop() {
		f := proto.Frame{Type: proto.TOutput, Payload: proto.EncodeOutput(off, data)}
		if _, err := a.frame(f); err != nil {
			b.Fatal(err)
		}
		off += int64(len(data))
	}
}
