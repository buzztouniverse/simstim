package term

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"
)

func collect(chunks ...string) []Token {
	var p Parser
	var out []Token
	for _, c := range chunks {
		p.Feed([]byte(c), func(t Token) {
			t.Raw = append([]byte(nil), t.Raw...)
			out = append(out, t)
		})
	}
	return out
}

func TestParserSplitsSequences(t *testing.T) {
	toks := collect("ab\x1b[?2004h", "cd\x1b]0;title\x07", "\x1b[3", "1mx")
	var kinds []Kind
	var raw []string
	for _, tk := range toks {
		kinds = append(kinds, tk.Kind)
		raw = append(raw, string(tk.Raw))
	}
	want := []string{"ab", "\x1b[?2004h", "cd", "\x1b]0;title\x07", "\x1b[31m", "x"}
	if strings.Join(raw, "|") != strings.Join(want, "|") {
		t.Fatalf("raw = %q", raw)
	}
	if kinds[1] != CSI || kinds[3] != OSC {
		t.Fatalf("kinds = %v", kinds)
	}
}

// The parser must never lose or reorder bytes, whatever the chunking.
func TestParserPreservesBytes(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	pieces := []string{"hello ", "\x1b[?1049h", "\x1b[>1u", "\x1b]11;?\x1b\\", "\x1bP+q544e\x1b\\", "\x1b=", "日本語", "\x1b[38;2;10;20;30m", "\r\n", "\x07"}
	for round := 0; round < 200; round++ {
		var stream []byte
		for i := 0; i < 50; i++ {
			stream = append(stream, pieces[rng.Intn(len(pieces))]...)
		}
		var p Parser
		var got []byte
		for i := 0; i < len(stream); {
			n := 1 + rng.Intn(9)
			if i+n > len(stream) {
				n = len(stream) - i
			}
			p.Feed(stream[i:i+n], func(t Token) { got = append(got, t.Raw...) })
			i += n
		}
		got = append(got, p.Flush()...)
		if !bytes.Equal(got, stream) {
			t.Fatalf("round %d: bytes changed\n got %q\nwant %q", round, got, stream)
		}
	}
}

func TestTrackerKittyAndModes(t *testing.T) {
	tr := NewTracker()
	tr.Feed([]byte("\x1b[>1u\x1b[?2004h\x1b[?1000;1006h\x1b[?1049h\x1b[>3u\x1b[?25l\x1b[2 q"))
	m := tr.M
	if !m.AltScreen || len(m.KittyMain) != 1 || len(m.KittyAlt) != 1 || m.KittyAlt[0] != 3 {
		t.Fatalf("modes: %+v", m)
	}
	if !m.Private[2004] || !m.Private[1000] || !m.Private[1006] || m.Private[25] || m.CursorShape != 2 {
		t.Fatalf("private modes: %+v", m.Private)
	}
	reset := string(m.ResetSequence())
	for _, want := range []string{"\x1b[<1u\x1b[?1049l\x1b[<1u", "\x1b[?2004l", "\x1b[?1000l", "\x1b[?25h", "\x1b[0 q"} {
		if !strings.Contains(reset, want) {
			t.Errorf("reset lacks %q: %q", want, reset)
		}
	}
	if strings.Contains(reset, "\x1bc") {
		t.Fatal("reset must never contain RIS")
	}
	restore := string(m.RestoreSequence())
	for _, want := range []string{"\x1b[>1u\x1b[?1049h\x1b[>3u", "\x1b[?2004h", "\x1b[?25l", "\x1b[2 q"} {
		if !strings.Contains(restore, want) {
			t.Errorf("restore lacks %q: %q", want, restore)
		}
	}
}

func TestTrackerPopAndLeaveAlt(t *testing.T) {
	tr := NewTracker()
	tr.Feed([]byte("\x1b[>1u\x1b[>5u\x1b[<u\x1b[?1049h\x1b[>1u\x1b[?1049l"))
	if tr.M.AltScreen || len(tr.M.KittyMain) != 1 || tr.M.KittyMain[0] != 1 {
		t.Fatalf("modes: %+v", tr.M)
	}
	tr.Feed([]byte("\x1bc"))
	if len(tr.M.KittyMain) != 0 {
		t.Fatal("RIS should reset the tracker")
	}
}

func TestSanitizerDropsQueries(t *testing.T) {
	in := "\x1b]99;i=opentui-notifications:p=?;\x1b\\a\x1b[c\x1b[>c\x1b[6n\x1b[>q\x1b[?u\x1b]11;?\x07\x1b]52;c;?\x1b\\\x1bP+q544e\x1b\\\x1b[?2004$p\x1b[14tb" +
		"\x1b[>4n\x1b[22;0t\x1b[31mc\x1b]0;keep\x07"
	var s Sanitizer
	got := string(s.Feed([]byte(in)))
	want := "ab\x1b[>4n\x1b[22;0t\x1b[31mc\x1b]0;keep\x07"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestInts(t *testing.T) {
	cases := map[string][]int{"": nil, "1;2": {1, 2}, ";5": {-1, 5}, "118:86;5": {118, 5}, "4": {4}}
	for in, want := range cases {
		got := Ints([]byte(in))
		if len(got) != len(want) {
			t.Fatalf("Ints(%q) = %v", in, got)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("Ints(%q) = %v", in, got)
			}
		}
	}
}
