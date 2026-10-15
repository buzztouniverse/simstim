package term

import "bytes"

// Sanitizer removes terminal queries from replayed history. Replay is history,
// not a live conversation: the local terminal must not answer old questions,
// or its answers would reach the agent as typed input.
type Sanitizer struct {
	p Parser
}

// Feed returns p without queries. Incomplete sequences are kept for the next call.
func (s *Sanitizer) Feed(p []byte) []byte {
	var out bytes.Buffer
	s.p.Feed(p, func(t Token) {
		if !IsQuery(t) {
			out.Write(t.Raw)
		}
	})
	return out.Bytes()
}

// Flush returns any buffered incomplete sequence, unchanged.
func (s *Sanitizer) Flush() []byte { return s.p.Flush() }

// IsQuery reports whether a token asks the terminal for a reply.
func IsQuery(t Token) bool {
	switch t.Kind {
	case CSI:
		switch t.Final {
		case 'c': // DA1, DA2, DA3
			return len(t.Intermediate) == 0
		case 'n': // DSR and private DSR; CSI > 4 n is a mode reset, not a query
			return t.Prefix == 0 || t.Prefix == '?'
		case 'q': // XTVERSION
			return t.Prefix == '>' && len(t.Intermediate) == 0
		case 'u': // kitty keyboard query
			return t.Prefix == '?'
		case 'p': // DECRQM
			return bytes.Equal(t.Intermediate, []byte("$"))
		case 't': // window reports; 22 and 23 are the title stack, which we keep
			v := Ints(t.Params)
			if len(v) > 0 {
				switch v[0] {
				case 11, 13, 14, 15, 16, 18, 19, 20, 21:
					return true
				}
			}
		}
		return false
	case OSC:
		// Colour, clipboard, status and pointer queries end with "?".
		if bytes.HasSuffix(t.Payload, []byte("?")) || bytes.Contains(t.Payload, []byte(";?;")) {
			return true
		}
		// Kitty's notification protocol asks for capabilities with p=? in its metadata.
		if rest, ok := bytes.CutPrefix(t.Payload, []byte("99;")); ok {
			meta, _, _ := bytes.Cut(rest, []byte(";"))
			return bytes.Contains(meta, []byte("p=?"))
		}
		return false
	case DCS:
		// XTGETTCAP (DCS + q) and DECRQSS (DCS $ q).
		return bytes.HasPrefix(t.Payload, []byte("+q")) || bytes.HasPrefix(t.Payload, []byte("$q"))
	case StrSeq:
		// Kitty graphics queries (APC G with a=q).
		return bytes.HasPrefix(t.Payload, []byte("G")) && bytes.Contains(t.Payload, []byte("a=q"))
	default:
		return false
	}
}
