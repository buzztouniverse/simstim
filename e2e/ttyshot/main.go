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
package main

import (
	"bufio"
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

func main() {
	cols := flag.Int("cols", 110, "columns")
	rows := flag.Int("rows", 32, "rows")
	script := flag.String("script", "", "script file (default: stdin)")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: ttyshot [-cols n] [-rows n] [-script f] -- command [args…]")
		os.Exit(2)
	}
	cmd := exec.Command(flag.Arg(0), flag.Args()[1:]...)
	cmd.Env = os.Environ()
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(*cols), Rows: uint16(*rows)})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	emu := vt.NewSafeEmulator(*cols, *rows)
	// Replies to the program's queries go back to it, like a real terminal.
	go func() { _, _ = io.Copy(ptmx, emu) }()
	var mu sync.Mutex
	go func() {
		buf := make([]byte, 64<<10)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				mu.Lock()
				_, _ = emu.Write(buf[:n])
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	screen := func() string {
		mu.Lock()
		defer mu.Unlock()
		var b strings.Builder
		for y := 0; y < emu.Height(); y++ {
			var line strings.Builder
			for x := 0; x < emu.Width(); x++ {
				if c := emu.CellAt(x, y); c != nil && c.Content != "" {
					line.WriteString(c.Content)
				} else if c == nil || c.Width != 0 {
					line.WriteByte(' ')
				}
			}
			b.WriteString(strings.TrimRight(line.String(), " ") + "\n")
		}
		return b.String()
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
	failed := false
	sc := bufio.NewScanner(in)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		verb, rest, _ := strings.Cut(line, " ")
		switch verb {
		case "send":
			s, err := strconv.Unquote(strings.TrimSpace(rest))
			if err != nil {
				fmt.Fprintf(os.Stderr, "bad send %q: %v\n", rest, err)
				os.Exit(2)
			}
			_, _ = ptmx.Write([]byte(s))
		case "sleep":
			d, _ := time.ParseDuration(strings.TrimSpace(rest))
			time.Sleep(d)
		case "wait", "expect":
			q, tail, _ := strings.Cut(strings.TrimSpace(rest), "\" ")
			pat, err := strconv.Unquote(strings.TrimSuffix(q, "\"") + "\"")
			if err != nil {
				fmt.Fprintf(os.Stderr, "bad pattern %q\n", rest)
				os.Exit(2)
			}
			re := regexp.MustCompile(pat)
			timeout := 10 * time.Second
			if verb == "expect" {
				timeout = 0
			}
			if d, err := time.ParseDuration(strings.TrimSpace(tail)); err == nil {
				timeout = d
			}
			deadline := time.Now().Add(timeout)
			for !re.MatchString(screen()) && time.Now().Before(deadline) {
				time.Sleep(50 * time.Millisecond)
			}
			if !re.MatchString(screen()) {
				fmt.Printf("FAIL %s %q\n%s", verb, pat, screen())
				failed = true
			} else if verb == "expect" {
				fmt.Printf("PASS %q\n", pat)
			}
		case "shot":
			fmt.Print("----\n" + screen() + "----\n")
		}
	}
	_ = cmd.Process.Kill()
	if failed {
		os.Exit(1)
	}
}
