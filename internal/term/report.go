package term

// IsReport reports whether a token in a terminal's input is something the
// terminal sent by itself, not a key a person pressed: an answer to a query,
// a focus change, or another report. Mouse events are not reports, since a
// person made them.
func IsReport(t Token) bool {
	switch t.Kind {
	case OSC, DCS, StrSeq:
		// No key sends these; terminals use them to answer queries.
		return true
	case CSI:
		return isCSIReport(t)
	}
	return false
}

func isCSIReport(t Token) bool {
	switch t.Final {
	case 'c': // device attributes: CSI ? … c and CSI > … c
		return t.Prefix == '?' || t.Prefix == '>'
	case 'R': // cursor position (CSI row ; col R)
		return t.Prefix == 0 || t.Prefix == '?'
	case 'n': // status reports
		return true
	case 'y': // mode reports: CSI ? … $ y
		return len(t.Intermediate) > 0 && t.Intermediate[0] == '$'
	case 'u': // the kitty keyboard flags: CSI ? flags u (a key has no '?')
		return t.Prefix == '?'
	case 't': // window size and position reports
		return t.Prefix == 0
	case 'I', 'O': // focus in and out
		return t.Prefix == 0 && len(t.Params) == 0
	}
	return false
}
