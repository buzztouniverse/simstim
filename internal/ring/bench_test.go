package ring

import (
	"bytes"
	"testing"
)

// The host side of the attach pump keeps each session's output in a ring,
// and a client that reconnects reads its missed bytes from it.

func BenchmarkWrite(b *testing.B) {
	r := New(4 << 20)
	data := bytes.Repeat([]byte("x"), 32<<10)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		r.Write(data)
	}
}

func BenchmarkReadFrom(b *testing.B) {
	r := New(4 << 20)
	data := bytes.Repeat([]byte("x"), 32<<10)
	for range 200 {
		r.Write(data)
	}
	from := r.End() - 1<<20
	b.SetBytes(1 << 20)
	b.ReportAllocs()
	for b.Loop() {
		if _, ok := r.ReadFrom(from); !ok {
			b.Fatal("offset gone")
		}
	}
}
