package clienttest

import (
	"bytes"
	"context"
	"io"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/iamlakshya/simstim/internal/client"
)

// EveryByte is every byte value, then n random bytes, so that a channel which
// changes any byte (CR/LF, UTF-8, NUL) fails.
func EveryByte(n int) []byte {
	b := make([]byte, 0, 256+n)
	for i := range 256 {
		b = append(b, byte(i))
	}
	r := rand.New(rand.NewPCG(1, 2))
	for range n {
		b = append(b, byte(r.IntN(256)))
	}
	return b
}

// RoundTrip sends bytes through `cat` over a transport and fails the test
// unless every byte comes back unchanged and the command ends well.
func RoundTrip(t *testing.T, tr client.Transport) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := tr.Start(ctx, "cat")
	if err != nil {
		t.Fatal(err)
	}
	data := EveryByte(300 << 10)
	go func() {
		// Several writes of odd sizes, as frames arrive.
		for off := 0; off < len(data); {
			end := min(off+1+off%9000, len(data))
			if _, err := p.Stdin().Write(data[off:end]); err != nil {
				break
			}
			off = end
		}
		_ = p.Stdin().Close()
	}()
	got, err := io.ReadAll(p.Stdout())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Wait(); err != nil {
		t.Fatalf("wait: %v (stderr %q)", err, p.Stderr())
	}
	if !bytes.Equal(got, data) {
		at := 0
		for at < len(got) && at < len(data) && got[at] == data[at] {
			at++
		}
		t.Fatalf("the bytes changed on the way: sent %d, got %d, first difference at %d", len(data), len(got), at)
	}
}
