package main

import (
	"strings"
	"testing"
	"time"
)

// A row that stays the same across frames is drawn once, with one span; a
// row that changes gets a group for each look it has.
func TestAnimatedSVGKeepsEachRowOnce(t *testing.T) {
	frames := []frame{
		{at: 0, rows: []string{"<text>title</text>\n", "<text>a</text>\n"}},
		{at: time.Second, rows: []string{"<text>title</text>\n", "<text>b</text>\n"}},
		{at: 2 * time.Second, rows: []string{"<text>title</text>\n", "<text>a</text>\n"}},
	}
	svg := animatedSVG(frames, 4*time.Second, 10, 2)
	if n := strings.Count(svg, "<text>title</text>"); n != 1 {
		t.Errorf("the unchanged row is drawn %d times, want 1", n)
	}
	if n := strings.Count(svg, "<text>a</text>"); n != 1 {
		t.Errorf("the row that comes back is drawn %d times, want 1", n)
	}
	// "a" shows from 0 to 25 %, hides, and shows again from 50 % to the end.
	if !strings.Contains(svg, "0.00%{opacity:1}25.00%{opacity:0}50.00%{opacity:1}") {
		t.Errorf("wrong spans for the row that comes back:\n%s", svg)
	}
	if !strings.Contains(svg, "prefers-reduced-motion") || !strings.Contains(svg, `class="p p0 l"`) {
		t.Error("reduced motion does not show the last frame")
	}
}

func TestWriteCast(t *testing.T) {
	path := t.TempDir() + "/x.cast"
	if err := writeCast(path, []castEvent{{at: 1500 * time.Millisecond, data: "hi\r\n"}}, 80, 24); err != nil {
		t.Fatal(err)
	}
}
