package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/vt"
)

// A recording keeps what a program draws between "record start" and
// "record save": the screen each time it changes, for an animated picture,
// and the raw output with its timing, for an asciicast.
type recording struct {
	mu     sync.Mutex
	start  time.Time
	frames []frame
	out    []castEvent
	stop   chan struct{}
	done   chan struct{}
}

type frame struct {
	at   time.Duration
	rows []string
}

type castEvent struct {
	at   time.Duration
	data string
}

// maxFrames keeps a picture small; a demo that changes more often keeps its
// first frames.
const maxFrames = 240

// startRecording samples the screen every 40 ms and keeps the frames that
// differ from the one before.
func startRecording(emu *vt.SafeEmulator, lock *sync.Mutex) *recording {
	r := &recording{start: time.Now(), stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(r.done)
		t := time.NewTicker(40 * time.Millisecond)
		defer t.Stop()
		last := ""
		for {
			lock.Lock()
			key := emu.Render()
			var rows []string
			if key != last {
				rows = svgRows(emu)
			}
			lock.Unlock()
			if key != last {
				last = key
				r.mu.Lock()
				if len(r.frames) < maxFrames {
					r.frames = append(r.frames, frame{at: time.Since(r.start), rows: rows})
				}
				r.mu.Unlock()
			}
			select {
			case <-r.stop:
				return
			case <-t.C:
			}
		}
	}()
	return r
}

// output keeps a chunk of the program's output for the cast.
func (r *recording) output(b []byte) {
	r.mu.Lock()
	r.out = append(r.out, castEvent{at: time.Since(r.start), data: string(b)})
	r.mu.Unlock()
}

// save stops the recording and writes an animated SVG, and an asciicast v2
// file when castPath is set. The last frame holds for hold before the loop
// starts again.
func (r *recording) save(svgPath, castPath string, cols, rows int, hold time.Duration) error {
	close(r.stop)
	<-r.done
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.frames) == 0 {
		return fmt.Errorf("record: no frames")
	}
	end := time.Since(r.start) + hold
	if err := os.WriteFile(svgPath, []byte(animatedSVG(r.frames, end, cols, rows)), 0o644); err != nil {
		return fmt.Errorf("record: %w", err)
	}
	if castPath == "" {
		return nil
	}
	return writeCast(castPath, r.out, cols, rows)
}

// animatedSVG shows each frame from its time until the next one's, in a loop
// of length end. Rows that stay the same are drawn once, and each distinct
// row shows during the spans when the screen has it. With reduced motion,
// only the last frame shows.
func animatedSVG(frames []frame, end time.Duration, cols, rows int) string {
	width, height := windowSize(cols, rows)
	total := end.Seconds()
	pct := func(d time.Duration) float64 { return 100 * d.Seconds() / total }
	type span struct{ from, to time.Duration }
	type piece struct {
		body  string
		spans []span
	}
	var pieces []*piece
	index := map[string]*piece{} // row number and body
	for i, f := range frames {
		next := end
		if i+1 < len(frames) {
			next = frames[i+1].at
		}
		from := f.at
		if i == 0 {
			from = 0
		}
		for y, body := range f.rows {
			if strings.TrimSpace(body) == "" {
				continue
			}
			key := fmt.Sprint(y, "\x00", body)
			p := index[key]
			if p == nil {
				p = &piece{body: body}
				index[key] = p
				pieces = append(pieces, p)
			}
			if n := len(p.spans); n > 0 && p.spans[n-1].to == from {
				p.spans[n-1].to = next // the row stays: one span
			} else {
				p.spans = append(p.spans, span{from, next})
			}
		}
	}
	lastRows := map[string]bool{}
	for y, body := range frames[len(frames)-1].rows {
		lastRows[fmt.Sprint(y, "\x00", body)] = true
	}
	var css, body strings.Builder
	fmt.Fprintf(&css, "g.p{opacity:0;animation:%.2fs steps(1,end) infinite}", total)
	fmt.Fprint(&css, "@media (prefers-reduced-motion:reduce){g.p{animation:none}g.l{opacity:1}}")
	for i, p := range pieces {
		fmt.Fprintf(&css, "@keyframes k%d{", i)
		if p.spans[0].from > 0 {
			css.WriteString("0%{opacity:0}")
		}
		for _, sp := range p.spans {
			fmt.Fprintf(&css, "%.2f%%{opacity:1}", pct(sp.from))
			if sp.to < end {
				fmt.Fprintf(&css, "%.2f%%{opacity:0}", pct(sp.to))
			}
		}
		fmt.Fprintf(&css, "}.p%d{animation-name:k%d}", i, i)
	}
	for i, p := range pieces {
		class := fmt.Sprintf("p p%d", i)
		for key, q := range index {
			if q == p && lastRows[key] {
				class += " l"
				break
			}
		}
		fmt.Fprintf(&body, "<g class=\"%s\">\n%s</g>\n", class, p.body)
	}
	return svgHead(width, height, css.String()) + body.String() + "</svg>\n"
}

// writeCast writes an asciicast v2 file: a header line, then one line per
// chunk of output.
func writeCast(path string, events []castEvent, cols, rows int) error {
	var b strings.Builder
	head, _ := json.Marshal(map[string]any{
		"version": 2, "width": cols, "height": rows,
		"env": map[string]string{"TERM": os.Getenv("TERM")},
	})
	b.Write(head)
	b.WriteByte('\n')
	for _, e := range events {
		line, _ := json.Marshal([]any{roundMS(e.at), "o", e.data})
		b.Write(line)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("record: %w", err)
	}
	return nil
}

func roundMS(d time.Duration) float64 { return float64(d.Milliseconds()) / 1000 }
