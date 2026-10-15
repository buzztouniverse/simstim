// Package keys finds the client's own keys in the user's input, in every way a
// terminal can encode them, and passes every other byte through unchanged.
//
// While attached, the client owns only two keys: detach (default Ctrl+\) and
// image paste (default Ctrl+V). A terminal may send them as legacy control bytes,
// as kitty keyboard protocol CSI u sequences (with event types, lock bits and
// alternate keys), or as xterm modifyOtherKeys sequences. Bracketed pastes are
// collected whole, so dropped file paths can be rewritten, and never scanned for keys.
package keys

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Modifier bits, as in the kitty keyboard protocol (value - 1).
const (
	ModShift = 1
	ModAlt   = 2
	ModCtrl  = 4
	ModSuper = 8
	lockBits = 64 | 128 // caps lock and num lock are ignored
)

// Key is one key with modifiers.
type Key struct {
	Code rune // the unshifted key, for example '\\' or 'v'
	Mods int  // ModCtrl and friends
}

// String formats a key as "ctrl+\".
func (k Key) String() string {
	var parts []string
	if k.Mods&ModCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	if k.Mods&ModAlt != 0 {
		parts = append(parts, "alt")
	}
	if k.Mods&ModShift != 0 {
		parts = append(parts, "shift")
	}
	if k.Mods&ModSuper != 0 {
		parts = append(parts, "super")
	}
	return strings.Join(append(parts, string(k.Code)), "+")
}

// Parse reads a key such as "ctrl+\" or "alt+v".
func Parse(s string) (Key, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return Key{}, fmt.Errorf("empty key")
	}
	var k Key
	parts := strings.Split(s, "+")
	// "ctrl++" means ctrl and the plus key.
	if strings.HasSuffix(s, "++") {
		parts = append(strings.Split(strings.TrimSuffix(s, "++"), "+"), "+")
	}
	last := parts[len(parts)-1]
	for _, p := range parts[:len(parts)-1] {
		switch p {
		case "ctrl", "control":
			k.Mods |= ModCtrl
		case "alt", "opt", "option", "meta":
			k.Mods |= ModAlt
		case "shift":
			k.Mods |= ModShift
		case "super", "cmd":
			k.Mods |= ModSuper
		default:
			return Key{}, fmt.Errorf("unknown modifier %q in %q", p, s)
		}
	}
	r := []rune(last)
	if len(r) != 1 {
		return Key{}, fmt.Errorf("the key in %q must be one character", s)
	}
	k.Code = r[0]
	return k, nil
}

// legacy returns the bytes a terminal sends for k without any keyboard protocol,
// or nil when there is no unique legacy encoding.
func (k Key) legacy() []byte {
	c := k.Code
	switch k.Mods {
	case ModCtrl:
		switch {
		case c >= 'a' && c <= 'z':
			return []byte{byte(c - 'a' + 1)}
		case c >= '@' && c <= '_': // @ [ \ ] ^ _
			return []byte{byte(c - '@')}
		case c == '?':
			return []byte{0x7f}
		}
	case ModAlt:
		if c < 0x80 {
			return []byte{0x1b, byte(c)}
		}
	case 0:
		return []byte(string(c))
	}
	return nil
}

// Kind of a piece of filtered input.
type Kind int

// Piece kinds.
const (
	Pass  Kind = iota // bytes to forward unchanged
	Match             // one of the watched keys
	Paste             // a complete bracketed paste
)

// Piece is one part of the filtered input.
type Piece struct {
	Kind Kind
	// Raw is the exact input bytes of this piece.
	Raw []byte
	// For Match: which watched key (its index), and whether this is a release or repeat event.
	Index   int
	Release bool
	Repeat  bool
	// For Paste: the pasted text, without the bracketing markers.
	Text []byte
}

// Filter splits input into pass-through bytes, watched keys and pastes.
type Filter struct {
	keys    []Key
	legacy  [][]byte
	buf     []byte
	inPaste bool
	paste   bytes.Buffer
}

// MaxPaste is the largest paste collected whole; larger pastes pass through as they come.
const MaxPaste = 1 << 20

var (
	pasteStart = []byte("\x1b[200~")
	pasteEnd   = []byte("\x1b[201~")
)

// New returns a filter that watches for the given keys.
func New(watch ...Key) *Filter {
	f := &Filter{keys: watch}
	for _, k := range watch {
		f.legacy = append(f.legacy, k.legacy())
	}
	return f
}

// Feed filters p. Bytes of an unfinished escape sequence are kept until the next
// call, or until Flush. Use Pending to know whether to arm a short flush timer.
func (f *Filter) Feed(p []byte) []Piece {
	f.buf = append(f.buf, p...)
	var out []Piece
	b := f.buf
	i, passStart := 0, 0
	emitPass := func(end int) {
		if end > passStart {
			out = append(out, Piece{Kind: Pass, Raw: append([]byte(nil), b[passStart:end]...)})
		}
	}
	for i < len(b) {
		if f.inPaste {
			j := bytes.Index(b[i:], pasteEnd)
			if j < 0 {
				// Keep all but a possible partial end marker.
				keep := len(b) - i
				if keep > len(pasteEnd)-1 {
					keep = len(pasteEnd) - 1
				}
				f.paste.Write(b[i : len(b)-keep])
				i = len(b) - keep
				if f.paste.Len() > MaxPaste {
					out = append(out, Piece{Kind: Pass, Raw: append(append([]byte(nil), pasteStart...), f.paste.Bytes()...)})
					f.paste.Reset()
					f.inPaste = false
					passStart = i
					// The rest of this paste, and its end marker, pass through as normal input.
				}
				break
			}
			f.paste.Write(b[i : i+j])
			text := append([]byte(nil), f.paste.Bytes()...)
			raw := append(append(append([]byte(nil), pasteStart...), text...), pasteEnd...)
			out = append(out, Piece{Kind: Paste, Raw: raw, Text: text})
			f.paste.Reset()
			f.inPaste = false
			i += j + len(pasteEnd)
			passStart = i
			continue
		}
		c := b[i]
		if c != 0x1b {
			if idx := f.matchLegacyByte(c); idx >= 0 {
				emitPass(i)
				out = append(out, Piece{Kind: Match, Raw: []byte{c}, Index: idx})
				i++
				passStart = i
				continue
			}
			i++
			continue
		}
		// ESC: a sequence, an Alt+key, or the Esc key itself.
		if i+1 >= len(b) {
			// A lone ESC at the end of a read is almost always the Esc key.
			// Keep it only briefly (see Pending/Flush) in case the rest follows.
			break
		}
		if bytes.HasPrefix(b[i:], pasteStart) {
			emitPass(i)
			f.inPaste = true
			i += len(pasteStart)
			passStart = i
			continue
		}
		if b[i+1] == '[' {
			n, complete, valid := csiLen(b[i:])
			if !complete {
				break // incomplete
			}
			if !valid {
				i += n // not a real CSI: pass it through untouched
				continue
			}
			seq := b[i : i+n]
			if idx, rel, rep, ok := f.matchCSI(seq); ok {
				emitPass(i)
				out = append(out, Piece{Kind: Match, Raw: append([]byte(nil), seq...), Index: idx, Release: rel, Repeat: rep})
				i += n
				passStart = i
				continue
			}
			i += n
			continue
		}
		// Legacy Alt+key: ESC followed by the key.
		if idx := f.matchLegacyAlt(b[i : i+2]); idx >= 0 {
			emitPass(i)
			out = append(out, Piece{Kind: Match, Raw: append([]byte(nil), b[i:i+2]...), Index: idx})
			i += 2
			passStart = i
			continue
		}
		i++
	}
	if f.inPaste {
		emitPass(passStart)
		// Bytes after passStart up to i were consumed into the paste buffer.
		rest := append([]byte(nil), b[i:]...)
		f.buf = append(f.buf[:0], rest...)
		return dropEmpty(out)
	}
	emitPass(i)
	rest := append([]byte(nil), b[i:]...)
	f.buf = append(f.buf[:0], rest...)
	return dropEmpty(out)
}

func dropEmpty(ps []Piece) []Piece {
	out := ps[:0]
	for _, p := range ps {
		if p.Kind == Pass && len(p.Raw) == 0 {
			continue
		}
		out = append(out, p)
	}
	return out
}

// Pending reports whether bytes are held back.
func (f *Filter) Pending() bool { return len(f.buf) > 0 && !f.inPaste }

// Flush returns held-back bytes as pass-through input.
func (f *Filter) Flush() []byte {
	if f.inPaste {
		return nil
	}
	b := append([]byte(nil), f.buf...)
	f.buf = f.buf[:0]
	return b
}

func (f *Filter) matchLegacyByte(c byte) int {
	for i, l := range f.legacy {
		if len(l) == 1 && l[0] == c && c < 0x20 {
			return i
		}
	}
	return -1
}

func (f *Filter) matchLegacyAlt(two []byte) int {
	for i, l := range f.legacy {
		if len(l) == 2 && bytes.Equal(l, two) {
			return i
		}
	}
	return -1
}

// csiLen measures the CSI sequence at b. complete is false when more bytes are
// needed. valid is false when the bytes are not a well-formed CSI; then n bytes
// should pass through untouched.
func csiLen(b []byte) (n int, complete, valid bool) {
	for j := 2; j < len(b); j++ {
		c := b[j]
		if c >= 0x40 && c <= 0x7e {
			return j + 1, true, true
		}
		if c < 0x20 || c > 0x7e {
			return j, true, false
		}
	}
	return 0, false, false
}

// matchCSI matches kitty CSI u and xterm modifyOtherKeys encodings of a watched key.
func (f *Filter) matchCSI(seq []byte) (idx int, release, repeat, ok bool) {
	if len(seq) < 3 {
		return
	}
	body := seq[2 : len(seq)-1]
	final := seq[len(seq)-1]
	if len(body) > 0 && (body[0] < '0' || body[0] > '9') {
		return 0, false, false, false // has a prefix such as ? > < =: not a key
	}
	fields := strings.Split(string(body), ";")
	var code, mods, event int
	switch final {
	case 'u':
		// CSI code[:shifted[:base]] ; mods[:event] [; text] u
		c := strings.Split(fields[0], ":")
		var err error
		if code, err = strconv.Atoi(c[0]); err != nil {
			return
		}
		mods, event = 1, 1
		if len(fields) > 1 && fields[1] != "" {
			m := strings.Split(fields[1], ":")
			if mods, err = strconv.Atoi(m[0]); err != nil {
				return
			}
			if len(m) > 1 {
				if event, err = strconv.Atoi(m[1]); err != nil {
					return
				}
			}
		}
	case '~':
		// CSI 27 ; mods ; code ~
		if len(fields) != 3 || fields[0] != "27" {
			return
		}
		var err error
		if mods, err = strconv.Atoi(fields[1]); err != nil {
			return
		}
		if code, err = strconv.Atoi(fields[2]); err != nil {
			return
		}
		event = 1
	default:
		return
	}
	m := (mods - 1) &^ lockBits
	for i, k := range f.keys {
		if rune(code) == k.Code && m == k.Mods {
			return i, event == 3, event == 2, true
		}
	}
	return
}
