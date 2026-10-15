package ring

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestRingKeepsNewestBytes(t *testing.T) {
	r := New(8)
	r.Write([]byte("abcdef"))
	if got, ok := r.ReadFrom(0); !ok || string(got) != "abcdef" {
		t.Fatalf("ReadFrom(0) = %q, %v", got, ok)
	}
	r.Write([]byte("ghij"))
	if r.Start() != 2 || r.End() != 10 {
		t.Fatalf("start %d end %d", r.Start(), r.End())
	}
	if _, ok := r.ReadFrom(1); ok {
		t.Fatal("offset 1 should have fallen out")
	}
	if got, _ := r.ReadFrom(2); string(got) != "cdefghij" {
		t.Fatalf("ReadFrom(2) = %q", got)
	}
	if got, _ := r.ReadFrom(10); len(got) != 0 {
		t.Fatalf("ReadFrom(end) = %q", got)
	}
}

func TestRingBigWrite(t *testing.T) {
	r := New(4)
	r.Write([]byte("0123456789"))
	if got, _ := r.ReadFrom(r.Start()); string(got) != "6789" || r.Start() != 6 {
		t.Fatalf("got %q start %d", got, r.Start())
	}
}

func TestRingMatchesStream(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	r := New(97)
	var all []byte
	for i := range 2000 {
		chunk := make([]byte, rng.Intn(150))
		rng.Read(chunk)
		r.Write(chunk)
		all = append(all, chunk...)
		off := r.Start() + int64(rng.Intn(int(r.End()-r.Start())+1))
		got, ok := r.ReadFrom(off)
		if !ok || !bytes.Equal(got, all[off:]) {
			t.Fatalf("step %d: mismatch at offset %d", i, off)
		}
	}
}
