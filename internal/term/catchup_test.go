package term

import "testing"

func TestCatchup(t *testing.T) {
	var c Catchup
	c.History(9)
	// History loses its queries; live output passes byte for byte.
	show, fresh := c.Feed(0, []byte("hist\x1b[cAB")) // bytes 0 to 9
	if string(show) != "histAB" || string(fresh) != "hist\x1b[cAB" || c.Offset != 9 {
		t.Errorf("history: show %q fresh %q offset %d", show, fresh, c.Offset)
	}
	show, _ = c.Feed(9, []byte("live\x1b[c")) // bytes 9 to 16
	if string(show) != "live\x1b[c" || c.Offset != 16 {
		t.Errorf("live: show %q offset %d", show, c.Offset)
	}
	// A frame the terminal already shows is skipped, and an overlap trimmed.
	if show, fresh := c.Feed(9, []byte("live")); show != nil || fresh != nil {
		t.Errorf("duplicate: %q %q", show, fresh)
	}
	show, fresh = c.Feed(13, []byte("\x1b[cXY")) // bytes 13 to 18
	if string(show) != "XY" || string(fresh) != "XY" || c.Offset != 18 {
		t.Errorf("overlap: show %q fresh %q offset %d", show, fresh, c.Offset)
	}
	// A frame that straddles the end of history: only its history part is cleaned.
	c.History(23)
	show, _ = c.Feed(18, []byte("ab\x1b[cd\x1b[c")) // history to 23, then live
	if string(show) != "abd\x1b[c" || c.Offset != 27 {
		t.Errorf("straddle: show %q offset %d", show, c.Offset)
	}
}
