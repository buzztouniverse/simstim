package host

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/muesli/cancelreader"
	xterm "golang.org/x/term"

	"github.com/iAmLakshya/simstim/internal/paths"
)

// The phone path (decision 0019) is a hidden, experimental helper command,
// `pick`, that a person runs from any SSH app: `ssh -t host <helper> pick`.
// It shows the user's sessions as a plain numbered menu, then attaches the
// terminal to the one picked until Ctrl+\ detaches. It is not in the help or
// the user docs until the owner launches it; docs/contributing/phone-path.md
// describes it.

// RunPick runs the menu on in and out until the user quits.
func RunPick(in, out *os.File) error {
	signal.Ignore(syscall.SIGQUIT) // Ctrl+\ at the menu must not stop the program
	r := bufio.NewReader(in)
	host, _ := os.Hostname()
	for {
		list, err := List()
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}
		sessions := pickable(list)
		if len(sessions) == 0 {
			fmt.Fprintf(out, "No sessions are running on %s.\n", host)
			return nil
		}
		cols, _ := ttySize(out)
		fmt.Fprint(out, menu(host, sessions, time.Now(), cols))
		i, ok := readChoice(r, out, sessions)
		if !ok {
			return nil
		}
		s := sessions[i]
		fmt.Fprintf(out, "Attaching to %s. Ctrl+\\ detaches.\n", s.Name)
		end, err := attachTerminal(r, in, out, s.ID)
		switch {
		case err != nil:
			fmt.Fprintf(out, "\r\nCould not attach to %s: %v\n\n", s.Name, err)
		case end.ended != nil:
			fmt.Fprintf(out, "\r\n%s ended (exit code %d).\n\n", s.Name, end.ended.ExitCode)
		default:
			fmt.Fprintf(out, "\r\nDetached. %s keeps running.\n\n", s.Name)
		}
	}
}

// pickable keeps the sessions whose holder runs, so they can be attached.
func pickable(list []SessionInfo) []SessionInfo {
	var out []SessionInfo
	for _, s := range list {
		if s.Alive {
			out = append(out, s)
		}
	}
	return out
}

// menu draws the numbered list in plain text, fitted to width columns.
func menu(host string, sessions []SessionInfo, now time.Time, width int) string {
	if width <= 0 {
		width = 80
	}
	// "nn  name  agent  state  age": everything but the name has a fixed
	// width. A phone held upright has no room for the agent.
	agentW := 7
	if width < 50 {
		agentW = 0
	}
	fixed := 2 + 2 + 11 + 2 + 3 + 2
	if agentW > 0 {
		fixed += agentW + 2
	}
	nameW := max(width-fixed-1, 6)
	var b strings.Builder
	fmt.Fprintf(&b, "\nSessions on %s\n", host)
	for i, s := range sessions {
		age := "-"
		if !s.CreatedAt.IsZero() {
			age = shortAge(now.Sub(s.CreatedAt))
		}
		agent := ""
		if agentW > 0 {
			agent = fmt.Sprintf("%-*s  ", agentW, clip(s.Agent, agentW))
		}
		fmt.Fprintf(&b, "%2d  %-*s  %s%-11s  %3s\n", i+1, nameW, clip(s.Name, nameW), agent,
			strings.ReplaceAll(s.State, "_", " "), age)
	}
	return b.String()
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// shortAge writes d in at most three cells: 42s, 7m, 3h, 12d (99d at most).
func shortAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return strconv.Itoa(int(d.Seconds())) + "s"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < 48*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h"
	}
	return strconv.Itoa(min(int(d.Hours()/24), 99)) + "d"
}

// readChoice asks until it gets a session or the user quits. ok is false to quit.
func readChoice(r *bufio.Reader, out io.Writer, sessions []SessionInfo) (i int, ok bool) {
	for {
		fmt.Fprintf(out, "Pick 1-%d, or q to quit: ", len(sessions))
		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			fmt.Fprintln(out)
			return 0, false
		}
		i, quit, perr := parseChoice(line, sessions)
		if quit {
			return 0, false
		}
		if perr == nil {
			return i, true
		}
		fmt.Fprintln(out, perr)
	}
}

// parseChoice reads a menu answer: a number, a session's ID or name, or q.
func parseChoice(line string, sessions []SessionInfo) (i int, quit bool, err error) {
	s := strings.TrimSpace(line)
	switch strings.ToLower(s) {
	case "q", "quit", "exit":
		return 0, true, nil
	case "":
		return 0, false, errors.New("type a number")
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n < 1 || n > len(sessions) {
			return 0, false, fmt.Errorf("pick a number from 1 to %d", len(sessions))
		}
		return n - 1, false, nil
	}
	found := -1
	for j, x := range sessions {
		if x.ID == s {
			return j, false, nil
		}
		if x.Name == s {
			if found >= 0 {
				return 0, false, fmt.Errorf("more than one session is named %s; pick its number", s)
			}
			found = j
		}
	}
	if found < 0 {
		return 0, false, fmt.Errorf("no session %s; pick a number from 1 to %d", s, len(sessions))
	}
	return found, false, nil
}

// attachTerminal attaches this terminal to session id, in raw mode, until the
// user detaches or the session ends. r holds anything typed after the choice.
func attachTerminal(r *bufio.Reader, in, out *os.File, id string) (pickEnd, error) {
	conn, err := net.DialTimeout("unix", paths.Socket(id), 3*time.Second)
	if err != nil {
		return pickEnd{}, errors.New("the session is not running")
	}
	defer conn.Close()
	if fd := int(in.Fd()); xterm.IsTerminal(fd) {
		old, err := xterm.MakeRaw(fd)
		if err != nil {
			return pickEnd{}, fmt.Errorf("raw mode: %w", err)
		}
		defer func() { _ = xterm.Restore(fd, old) }()
	}
	cr, err := cancelreader.NewReader(in)
	if err != nil {
		return pickEnd{}, fmt.Errorf("read the terminal: %w", err)
	}
	// Keys typed after the choice wait in r; they go first.
	var first []byte
	if n := r.Buffered(); n > 0 {
		b, _ := r.Peek(n)
		first = append(first, b...)
		_, _ = r.Discard(n)
	}
	input := make(chan []byte, 64)
	stop, done := make(chan struct{}), make(chan struct{})
	go readInput(first, cr, input, stop, done)
	defer func() {
		// Stop the reader before the menu reads the terminal again.
		cr.Cancel()
		close(stop)
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		_ = cr.Close()
	}()

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	p := &pickPump{conn: conn, out: out, winch: winch, size: func() (int, int) { return ttySize(out) }}
	return p.run(id, input)
}

// readInput sends what the user types to input, first then what r reads,
// until r fails or stop closes. It closes done when it returns.
func readInput(first []byte, r io.Reader, input chan<- []byte, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	defer close(input)
	send := func(b []byte) bool {
		select {
		case input <- append([]byte(nil), b...):
			return true
		case <-stop:
			return false
		}
	}
	if len(first) > 0 && !send(first) {
		return
	}
	buf := make([]byte, 32<<10)
	for {
		n, err := r.Read(buf)
		if n > 0 && !send(buf[:n]) {
			return
		}
		if err != nil {
			return
		}
	}
}

func ttySize(f *os.File) (cols, rows int) {
	c, r, err := xterm.GetSize(int(f.Fd()))
	if err != nil || c <= 0 || r <= 0 {
		return 80, 24
	}
	return c, r
}
