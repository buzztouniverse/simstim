package keys

import (
	"bytes"
	"math/rand"
	"testing"
)

var (
	detach = Key{Code: '\\', Mods: ModCtrl}
	paste  = Key{Code: 'v', Mods: ModCtrl}
)

func feedAll(f *Filter, chunks ...[]byte) []Piece {
	var out []Piece
	for _, c := range chunks {
		out = append(out, f.Feed(c)...)
	}
	if b := f.Flush(); len(b) > 0 {
		out = append(out, Piece{Kind: Pass, Raw: b})
	}
	return out
}

func joined(ps []Piece) []byte {
	var b []byte
	for _, p := range ps {
		b = append(b, p.Raw...)
	}
	return b
}

func TestMatchesEveryEncoding(t *testing.T) {
	cases := []struct {
		in      string
		idx     int
		release bool
	}{
		{"\x1c", 0, false},
		{"\x1b[92;5u", 0, false},
		{"\x1b[92;5:1u", 0, false},
		{"\x1b[92;5:3u", 0, true},
		{"\x1b[92;69u", 0, false},  // caps lock
		{"\x1b[92;133u", 0, false}, // num lock
		{"\x1b[92:124;5u", 0, false},
		{"\x1b[27;5;92~", 0, false},
		{"\x16", 1, false},
		{"\x1b[118;5u", 1, false},
		{"\x1b[27;5;118~", 1, false},
	}
	for _, c := range cases {
		f := New(detach, paste)
		ps := feedAll(f, []byte("a"+c.in+"b"))
		if len(ps) != 3 || ps[1].Kind != Match || ps[1].Index != c.idx || ps[1].Release != c.release {
			t.Errorf("%q: got %+v", c.in, ps)
		}
		if !bytes.Equal(joined(ps), []byte("a"+c.in+"b")) {
			t.Errorf("%q: bytes changed", c.in)
		}
	}
}

func TestPassesOtherKeys(t *testing.T) {
	for _, in := range []string{"\x1b[118;6u", "\x1b[97;5u", "\x01", "\x1b[13;2u", "\x1b[A", "\x1bOA", "\x1b[?1u", "\x1bv", "\x1b[27;6;92~"} {
		f := New(detach, paste)
		ps := feedAll(f, []byte(in))
		for _, p := range ps {
			if p.Kind != Pass {
				t.Errorf("%q: unexpected %+v", in, p)
			}
		}
		if !bytes.Equal(joined(ps), []byte(in)) {
			t.Errorf("%q: bytes changed to %q", in, joined(ps))
		}
	}
}

func TestPasteIsCollectedWhole(t *testing.T) {
	f := New(detach, paste)
	ps := feedAll(f, []byte("x\x1b[200~/Users/me/a.png\x16"), []byte(" b\x1b[20"), []byte("1~y"))
	if len(ps) != 3 || ps[1].Kind != Paste || string(ps[1].Text) != "/Users/me/a.png\x16 b" {
		t.Fatalf("got %+v", ps)
	}
	if string(joined(ps)) != "x\x1b[200~/Users/me/a.png\x16 b\x1b[201~y" {
		t.Fatalf("bytes changed: %q", joined(ps))
	}
}

func TestPendingEscape(t *testing.T) {
	f := New(detach, paste)
	ps := f.Feed([]byte("a\x1b"))
	if len(ps) != 1 || string(ps[0].Raw) != "a" || !f.Pending() {
		t.Fatalf("got %+v pending=%v", ps, f.Pending())
	}
	if string(f.Flush()) != "\x1b" {
		t.Fatal("flush should return the held ESC")
	}
}

// Whatever the chunking, the filter must keep every byte, in order,
// and find exactly the watched keys.
func TestSplitStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	type piece struct {
		s     string
		match bool
	}
	vocab := []piece{
		{"hello", false}, {"\x1b[A", false}, {"\x1b[13;2u", false}, {"日本", false}, {"\r", false},
		{"\x1b[92;5u", true}, {"\x1c", true}, {"\x1b[118;5:1u", true}, {"\x1b[27;5;118~", true},
		{"\x1b[200~pasted \x1c text\x1b[201~", false},
	}
	for round := 0; round < 2000; round++ {
		var stream []byte
		want := 0
		for i := 0; i < 20; i++ {
			p := vocab[rng.Intn(len(vocab))]
			stream = append(stream, p.s...)
			if p.match {
				want++
			}
		}
		f := New(detach, paste)
		var chunks [][]byte
		for i := 0; i < len(stream); {
			n := 1 + rng.Intn(12)
			if i+n > len(stream) {
				n = len(stream) - i
			}
			chunks = append(chunks, stream[i:i+n])
			i += n
		}
		ps := feedAll(f, chunks...)
		got := 0
		for _, p := range ps {
			if p.Kind == Match {
				got++
			}
		}
		if !bytes.Equal(joined(ps), stream) {
			t.Fatalf("round %d: bytes changed\n got %q\nwant %q", round, joined(ps), stream)
		}
		if got != want {
			t.Fatalf("round %d: %d matches, want %d", round, got, want)
		}
	}
}

func TestParse(t *testing.T) {
	for in, want := range map[string]Key{"ctrl+\\": detach, "Ctrl+V": paste, "alt+v": {Code: 'v', Mods: ModAlt}} {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := Parse("hyper+x"); err == nil {
		t.Error("expected an error for an unknown modifier")
	}
}

func FuzzFilter(f *testing.F) {
	for _, s := range []string{"a\x1c", "\x1b[92;5u", "\x1b[200~x\x1b[201~", "\x1b", "\x1b[", "\x1b[27;5;118~"} {
		f.Add([]byte(s), uint8(3))
	}
	f.Fuzz(func(t *testing.T, data []byte, step uint8) {
		n := int(step%16) + 1
		flt := New(detach, paste)
		var chunks [][]byte
		for i := 0; i < len(data); i += n {
			end := i + n
			if end > len(data) {
				end = len(data)
			}
			chunks = append(chunks, data[i:end])
		}
		ps := feedAll(flt, chunks...)
		// An unfinished paste is still held by the filter; that is fine.
		if flt.inPaste {
			return
		}
		if !bytes.Equal(joined(ps), data) {
			t.Fatalf("bytes changed: got %q want %q", joined(ps), data)
		}
	})
}
