// Command ttyshot runs a program in a pseudo-terminal, feeds it keys from a
// script, and prints the rendered screen at checkpoints. It lets tests see a
// TUI the way a person would, not as a raw byte stream.
//
// Script lines:
//
//	send "text with \r and \x1b escapes"   (a Go-quoted string)
//	wait "regex" [timeout]                 (wait until the screen matches)
//	sleep 500ms
//	shot                                   (print the screen)
//	expect "regex"                         (fail unless the screen matches)
//	save path                              (write the screen with colours: SVG if path ends in .svg, else ANSI)
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

// term is the program under test: its pseudo-terminal, the emulator that
// draws its screen, and an animation being recorded.
type term struct {
	ptmx       *os.File
	emu        *vt.SafeEmulator
	mu         sync.Mutex
	recMu      sync.Mutex
	rec        *recording
	cols, rows int
	failed     bool
}

// exitError ends ttyshot with code: 2 for a bad script, 1 for a failure.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func main() {
	cols := flag.Int("cols", 110, "columns")
	rows := flag.Int("rows", 32, "rows")
	script := flag.String("script", "", "script file (default: stdin)")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: ttyshot [-cols n] [-rows n] [-script f] -- command [args…]")
		os.Exit(2)
	}
	if *cols < 1 || *cols > 1000 || *rows < 1 || *rows > 1000 {
		fmt.Fprintln(os.Stderr, "ttyshot: -cols and -rows must be between 1 and 1000")
		os.Exit(2)
	}
	in := os.Stdin
	if *script != "" {
		f, err := os.Open(*script)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		in = f
	}
	cmd := exec.CommandContext(context.Background(), flag.Arg(0), flag.Args()[1:]...)
	cmd.Env = os.Environ()
	t, err := start(cmd, *cols, *rows)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = t.run(in)
	_ = cmd.Process.Kill()
	var ee *exitError
	switch {
	case errors.As(err, &ee):
		fmt.Fprintln(os.Stderr, ee.msg)
		os.Exit(ee.code)
	case err != nil:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	case t.failed:
		os.Exit(1)
	}
}

// start runs cmd in a pseudo-terminal of the given size, and draws its output.
func start(cmd *exec.Cmd, cols, rows int) (*term, error) {
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}) //nolint:gosec // G115: main checks both are at most 1000
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", cmd.Path, err)
	}
	t := &term{ptmx: ptmx, emu: vt.NewSafeEmulator(cols, rows), cols: cols, rows: rows}
	// Replies to the program's queries go back to it, like a real terminal.
	go func() { _, _ = io.Copy(ptmx, t.emu) }()
	go t.pump()
	return t, nil
}

// pump feeds the program's output to the emulator, and to a recording.
func (t *term) pump() {
	buf := make([]byte, 64<<10)
	for {
		n, err := t.ptmx.Read(buf)
		if n > 0 {
			t.mu.Lock()
			_, _ = t.emu.Write(buf[:n])
			t.mu.Unlock()
			t.recMu.Lock()
			if t.rec != nil {
				t.rec.output(buf[:n])
			}
			t.recMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// screen is the screen's text, one line a row, without trailing spaces.
func (t *term) screen() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var b strings.Builder
	for y := range t.emu.Height() {
		var line strings.Builder
		for x := range t.emu.Width() {
			if c := t.emu.CellAt(x, y); c != nil && c.Content != "" {
				line.WriteString(c.Content)
			} else if c == nil || c.Width != 0 {
				line.WriteByte(' ')
			}
		}
		b.WriteString(strings.TrimRight(line.String(), " ") + "\n")
	}
	return b.String()
}

// run follows the script, one line at a time.
func (t *term) run(in io.Reader) error {
	sc := bufio.NewScanner(in)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		verb, rest, _ := strings.Cut(line, " ")
		if err := t.do(verb, rest); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read the script: %w", err)
	}
	return nil
}

func (t *term) do(verb, rest string) error {
	switch verb {
	case "send":
		s, err := strconv.Unquote(strings.TrimSpace(rest))
		if err != nil {
			return &exitError{2, fmt.Sprintf("bad send %q: %v", rest, err)}
		}
		_, _ = t.ptmx.Write([]byte(s))
	case "type":
		return t.typeText(rest)
	case "sleep":
		d, _ := time.ParseDuration(strings.TrimSpace(rest))
		time.Sleep(d)
	case "wait", "expect":
		return t.wait(verb, rest)
	case "shot":
		fmt.Print("----\n" + t.screen() + "----\n")
	case "record":
		return t.record(rest)
	case "save":
		return t.save(strings.TrimSpace(rest))
	}
	return nil
}

// typeText types "text" [delay] one key at a time, as a person types.
func (t *term) typeText(rest string) error {
	q, tail, _ := strings.Cut(strings.TrimSpace(rest), "\" ")
	text, err := strconv.Unquote(strings.TrimSuffix(q, "\"") + "\"")
	if err != nil {
		return &exitError{2, fmt.Sprintf("bad type %q: %v", rest, err)}
	}
	delay := 70 * time.Millisecond
	if d, err := time.ParseDuration(strings.TrimSpace(tail)); err == nil {
		delay = d
	}
	for _, r := range text {
		_, _ = t.ptmx.Write([]byte(string(r)))
		time.Sleep(delay)
	}
	return nil
}

// wait waits until the screen matches "regex" [timeout]; expect checks once.
func (t *term) wait(verb, rest string) error {
	q, tail, _ := strings.Cut(strings.TrimSpace(rest), "\" ")
	pat, err := strconv.Unquote(strings.TrimSuffix(q, "\"") + "\"")
	if err != nil {
		return &exitError{2, fmt.Sprintf("bad pattern %q", rest)}
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return &exitError{2, fmt.Sprintf("bad pattern %q: %v", pat, err)}
	}
	timeout := 10 * time.Second
	if verb == "expect" {
		timeout = 0
	}
	if d, err := time.ParseDuration(strings.TrimSpace(tail)); err == nil {
		timeout = d
	}
	deadline := time.Now().Add(timeout)
	for !re.MatchString(t.screen()) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	switch {
	case !re.MatchString(t.screen()):
		fmt.Printf("FAIL %s %q\n%s", verb, pat, t.screen())
		t.failed = true
	case verb == "expect":
		fmt.Printf("PASS %q\n", pat)
	}
	return nil
}

// record handles record start, and record save <picture.svg> [<cast>] [hold].
func (t *term) record(rest string) error {
	args := strings.Fields(rest)
	switch {
	case len(args) == 1 && args[0] == "start":
		t.recMu.Lock()
		t.rec = startRecording(t.emu, &t.mu)
		t.recMu.Unlock()
	case len(args) >= 2 && args[0] == "save" && t.rec != nil:
		cast, hold := "", 2*time.Second
		for _, a := range args[2:] {
			if d, err := time.ParseDuration(a); err == nil {
				hold = d
			} else {
				cast = a
			}
		}
		t.recMu.Lock()
		r := t.rec
		t.rec = nil
		t.recMu.Unlock()
		return r.save(args[1], cast, t.cols, t.rows, hold)
	default:
		return &exitError{2, fmt.Sprintf("bad record %q", rest)}
	}
	return nil
}

// save writes the screen with its colours and styles: an SVG image for a
// path ending in .svg, otherwise ANSI text.
func (t *term) save(path string) error {
	t.mu.Lock()
	styled := t.emu.Render()
	if strings.HasSuffix(path, ".svg") {
		styled = svgScreen(t.emu)
	}
	t.mu.Unlock()
	if err := os.WriteFile(path, []byte(styled), 0o644); err != nil {
		return fmt.Errorf("save the screen: %w", err)
	}
	return nil
}
