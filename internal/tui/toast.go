package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

type toastKind int

const (
	toastInfo toastKind = iota
	toastSuccess
	toastWarning
	toastError
)

// maxToasts is how many toasts show at once. More wait in a queue.
const maxToasts = 3

type toast struct {
	kind       toastKind
	text, hint string
	life       time.Duration
	until      time.Time
}

type toastTickMsg struct{}

// toastStack holds the visible toasts and the queue behind them.
type toastStack struct {
	shown []toast
	queue []toast
}

func lifeOf(k toastKind) time.Duration {
	if k == toastError {
		return toastErrorLife
	}
	return toastLife
}

// push adds a toast. A repeat of a visible one extends it instead.
func (t *toastStack) push(now time.Time, kind toastKind, text, hint string) {
	life := lifeOf(kind)
	for i := range t.shown {
		if t.shown[i].text == text {
			t.shown[i].until = now.Add(life)
			return
		}
	}
	for _, q := range t.queue {
		if q.text == text {
			return
		}
	}
	nt := toast{kind: kind, text: text, hint: hint, life: life, until: now.Add(life)}
	if len(t.shown) < maxToasts {
		t.shown = append(t.shown, nt)
	} else {
		t.queue = append(t.queue, nt)
	}
}

// expire drops the toasts whose time is up and shows queued ones. It reports
// whether anything is left to time.
func (t *toastStack) expire(now time.Time) bool {
	kept := t.shown[:0]
	for _, x := range t.shown {
		if now.Before(x.until) {
			kept = append(kept, x)
		}
	}
	t.shown = kept
	for len(t.shown) < maxToasts && len(t.queue) > 0 {
		q := t.queue[0]
		t.queue = t.queue[1:]
		q.until = now.Add(q.life)
		t.shown = append(t.shown, q)
	}
	return len(t.shown) > 0
}

// next is the time until the first visible toast expires.
func (t *toastStack) next(now time.Time) (time.Duration, bool) {
	if len(t.shown) == 0 {
		return 0, false
	}
	first := t.shown[0].until
	for _, x := range t.shown[1:] {
		if x.until.Before(first) {
			first = x.until
		}
	}
	return max(first.Sub(now), 10*time.Millisecond), true
}

func (m *Model) toast(kind toastKind, text string) tea.Cmd {
	m.toasts.push(m.now(), kind, text, "")
	return m.after(lifeOf(kind), toastTickMsg{})
}

// toastErr shows an error with its "Try:" line.
func (m *Model) toastErr(err error) tea.Cmd {
	msg, hint := errText(err)
	m.toasts.push(m.now(), toastError, sentence(msg), hint)
	return m.after(toastErrorLife, toastTickMsg{})
}

// toastLayer draws the toasts in one box at the bottom right, above the
// bottom border.
func (m *Model) toastLayer() (layer, bool) {
	if len(m.toasts.shown) == 0 || m.width < 20 {
		return layer{}, false
	}
	maxW := min(m.width-4, 64)
	var lines []string
	bs := boxNormal
	for _, t := range m.toasts.shown {
		glyph, st := glyphInfo, sInfo
		switch t.kind {
		case toastSuccess:
			glyph, st = glyphOK, sSuccess
		case toastWarning:
			glyph, st = glyphNeedsInput, sWarning
		case toastError:
			glyph, st = glyphFail, sDanger
			bs = boxDanger
		}
		for i, l := range wrap(t.text, maxW-6) {
			lead := "   "
			if i == 0 {
				lead = " " + st.Render(glyph) + " "
			}
			lines = append(lines, lead+l)
		}
		if t.hint != "" {
			for i, l := range wrap(t.hint, maxW-11) {
				lead := "   " + sPrimary.Render("Try:") + " "
				if i > 0 {
					lead = "        "
				}
				lines = append(lines, lead+l)
			}
		}
	}
	w := 0
	for _, l := range lines {
		w = max(w, cells(l))
	}
	w = min(w+3, maxW)
	h := len(lines) + 2
	b := box(w, h, "", "", "", lines, bs)
	return layer{s: b, x: max(m.width-w-2, 0), y: max(m.height-h-1, 0)}, true
}
