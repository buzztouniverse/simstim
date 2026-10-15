package keys

import (
	"bytes"
	"testing"
)

// The input side of the attach pump: every key the user types goes through
// the filter, which watches for the detach and paste keys.

func BenchmarkFilterTyping(b *testing.B) {
	f := New(Key{Code: '\\', Mods: ModCtrl}, Key{Code: 'v', Mods: ModCtrl})
	// Plain text, then the same keys in the kitty keyboard protocol.
	in := []byte("go test ./...\r")
	kitty := []byte("\x1b[103u\x1b[111u\x1b[13u\x1b[97;5u\x1b[97;5:3u")
	b.SetBytes(int64(len(in) + len(kitty)))
	b.ReportAllocs()
	for b.Loop() {
		_ = f.Feed(in)
		_ = f.Feed(kitty)
	}
}

func BenchmarkFilterPaste(b *testing.B) {
	f := New(Key{Code: '\\', Mods: ModCtrl})
	paste := append(append([]byte("\x1b[200~"), bytes.Repeat([]byte("pasted text "), 5000)...), "\x1b[201~"...)
	b.SetBytes(int64(len(paste)))
	b.ReportAllocs()
	for b.Loop() {
		_ = f.Feed(paste)
	}
}
