package tui

import (
	"slices"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// input is a one-line text field. It draws its own cursor as a reverse-video
// cell, so it needs no blinking and no terminal cursor.
type input struct {
	val         []rune
	pos         int
	placeholder string
}

func newInput(value, placeholder string) input {
	r := []rune(value)
	return input{val: r, pos: len(r), placeholder: placeholder}
}

func (in *input) value() string { return string(in.val) }

func (in *input) set(v string) {
	in.val = []rune(v)
	in.pos = len(in.val)
}

// insert adds text at the cursor. Control characters and line breaks are
// dropped, so a paste stays on one line.
func (in *input) insert(s string) {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	r := []rune(s)
	in.val = slices.Insert(in.val, in.pos, r...)
	in.pos += len(r)
}

// key edits the field. It reports whether the key was used.
func (in *input) key(k tea.KeyPressMsg) bool {
	switch k.String() {
	case "left", "ctrl+b":
		if in.pos > 0 {
			in.pos--
		}
	case "right", "ctrl+f":
		if in.pos < len(in.val) {
			in.pos++
		}
	case "home", "ctrl+a":
		in.pos = 0
	case "end", "ctrl+e":
		in.pos = len(in.val)
	case "backspace", "ctrl+h":
		if in.pos > 0 {
			in.val = slices.Delete(in.val, in.pos-1, in.pos)
			in.pos--
		}
	case "delete", "ctrl+d":
		if in.pos < len(in.val) {
			in.val = slices.Delete(in.val, in.pos, in.pos+1)
		}
	case "ctrl+u":
		in.val = slices.Delete(in.val, 0, in.pos)
		in.pos = 0
	case "ctrl+k":
		in.val = in.val[:in.pos]
	case "ctrl+w", "alt+backspace":
		start := in.pos
		for start > 0 && in.val[start-1] == ' ' {
			start--
		}
		for start > 0 && in.val[start-1] != ' ' && in.val[start-1] != '/' {
			start--
		}
		in.val = slices.Delete(in.val, start, in.pos)
		in.pos = start
	default:
		if k.Text == "" || k.Mod&(tea.ModCtrl|tea.ModAlt|tea.ModSuper|tea.ModMeta|tea.ModHyper) != 0 {
			return false
		}
		in.insert(k.Text)
	}
	return true
}

// view draws the field in w cells. A focused field shows its cursor.
func (in *input) view(w int, focused bool) string {
	if w <= 0 {
		return ""
	}
	if len(in.val) == 0 {
		ph := []rune(trunc(in.placeholder, w))
		if !focused {
			return sFaint.Render(string(ph))
		}
		if len(ph) == 0 {
			return sPlain.Reverse(true).Render(" ")
		}
		return sPlain.Reverse(true).Render(string(ph[:1])) + sFaint.Render(string(ph[1:]))
	}
	start := 0
	if focused && in.pos >= w {
		start = in.pos - w + 1
	}
	end := min(len(in.val), start+w)
	if !focused {
		return string(in.val[start:end])
	}
	before := string(in.val[start:in.pos])
	cur, after := " ", ""
	if in.pos < len(in.val) {
		cur = string(in.val[in.pos])
		if in.pos+1 < end {
			after = string(in.val[in.pos+1 : end])
		}
	}
	return before + sPlain.Reverse(true).Render(cur) + after
}
