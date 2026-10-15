package term

import "testing"

func TestIsReport(t *testing.T) {
	reports := []string{
		"\x1b[?62;22c", "\x1b[>1;10;0c", "\x1b[12;40R", "\x1b[0n", "\x1b[?2026;2$y", "\x1b[?1u",
		"\x1b[8;24;80t", "\x1b[I", "\x1b[O", "\x1b]11;rgb:0000/0000/0000\x1b\\", "\x1bP>|ghostty 1.2\x1b\\",
		"\x1b_Gi=1;OK\x1b\\",
	}
	keys := []string{
		"a", "\r", "\x1b[A", "\x1b[13;2u", "\x1b[97;5u", "\x1b[200~", "\x1b[<0;10;5M", "\x1bx", "\x1b[3~", "\x7f",
	}
	check := func(in string, want bool) {
		var ps Parser
		n := 0
		ps.Feed([]byte(in), func(tok Token) {
			n++
			if got := IsReport(tok); got != want {
				t.Errorf("IsReport(%q) = %v, want %v", in, got, want)
			}
		})
		if n == 0 {
			t.Errorf("%q gave no token", in)
		}
	}
	for _, r := range reports {
		check(r, true)
	}
	for _, k := range keys {
		check(k, false)
	}
}
