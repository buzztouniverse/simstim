package host

import "testing"

func TestScannerSignals(t *testing.T) {
	cases := map[string][]outSignal{
		"plain text":                         nil,
		"\x07":                               {sigBell},
		"\x1b]9;Build finished\x07":          {sigNotify},
		"\x1b]9;9;/home/me\x1b\\":            nil, // ConEmu: report the folder
		"\x1b]9;4;1;50\x07":                  {sigProgressBusy},
		"\x1b]9;4;0\x07":                     {sigProgressDone},
		"\x1b]777;notify;Title;Body\x1b\\":   {sigNotify},
		"\x1b]99;i=x:p=?;\x1b\\":             nil, // kitty: capability query
		"\x1b]99;i=1:d=0;Hello\x1b\\":        {sigNotify},
		"\x1b]0;title with \x07":             nil, // BEL ends the OSC, it is not a bell
		"\x1b[31mred\x1b[0m\x1b]8;;url\x07x": nil,
	}
	for in, want := range cases {
		var s scanner
		var got []outSignal
		// Feed one byte at a time, to check that state carries across chunks.
		for i := 0; i < len(in); i++ {
			s.feed([]byte{in[i]}, func(x outSignal) { got = append(got, x) })
		}
		if len(got) != len(want) {
			t.Errorf("%q: got %v, want %v", in, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%q: got %v, want %v", in, got, want)
			}
		}
	}
}
