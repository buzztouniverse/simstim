package host

// scanner watches a session's output for signals, without changing it.
// It finds the bell, desktop notifications (OSC 9 and OSC 777) and progress
// reports (OSC 9;4). It keeps its state across chunks.
type scanner struct {
	state   int
	osc     []byte
	escSeen bool
}

const (
	sNormal = iota
	sEsc
	sOSC
	sOSCEsc
	sOther // inside CSI or another sequence we skip
)

// outSignal is something the scanner found.
type outSignal int

const (
	sigBell outSignal = iota + 1
	sigNotify
	sigProgressBusy
	sigProgressDone
)

const maxOSC = 512

// feed scans p and calls fn for each outSignal.
func (s *scanner) feed(p []byte, fn func(outSignal)) {
	for _, c := range p {
		switch s.state {
		case sNormal:
			switch c {
			case 0x1b:
				s.state = sEsc
			case 0x07:
				fn(sigBell)
			}
		case sEsc:
			switch c {
			case ']':
				s.state = sOSC
				s.osc = s.osc[:0]
			case '[', 'P', '_', '^', 'X':
				s.state = sOther
			default:
				s.state = sNormal
			}
		case sOther:
			// CSI ends with a byte in 0x40–0x7e. DCS, APC, PM and SOS end with ST (ESC \) or BEL.
			if c == 0x1b {
				s.state = sEsc
			} else if c == 0x07 || (c >= 0x40 && c <= 0x7e) {
				s.state = sNormal
			}
		case sOSC:
			switch c {
			case 0x07:
				s.endOSC(fn)
			case 0x1b:
				s.state = sOSCEsc
			default:
				if len(s.osc) < maxOSC {
					s.osc = append(s.osc, c)
				}
			}
		case sOSCEsc:
			if c == '\\' {
				s.endOSC(fn)
			} else {
				s.state = sEsc
				s.feed([]byte{c}, fn)
			}
		}
	}
}

func (s *scanner) endOSC(fn func(outSignal)) {
	s.state = sNormal
	o := s.osc
	switch {
	case hasPrefix(o, "9;4;"):
		// OSC 9;4;state;progress. State 0 means done.
		if len(o) > 4 && o[4] == '0' {
			fn(sigProgressDone)
		} else {
			fn(sigProgressBusy)
		}
	case hasPrefix(o, "9;"), hasPrefix(o, "777;notify"), hasPrefix(o, "99;"):
		fn(sigNotify)
	}
}

func hasPrefix(b []byte, p string) bool {
	return len(b) >= len(p) && string(b[:len(p)]) == p
}
