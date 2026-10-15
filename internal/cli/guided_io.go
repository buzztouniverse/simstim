package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/muesli/cancelreader"

	"github.com/iamlakshya/simstim/internal/onboard"
)

// The guided setup's prompts read one line at a time, straight from the
// terminal, with no reader left behind: an attached session needs every
// byte of stdin. Ctrl+C ends the context, and a read waiting for a line
// returns at once.

// readLine reads one line from in. On a terminal the read can be cancelled
// by ctx; other readers are read a byte at a time, so nothing is buffered
// past the line.
func readLine(ctx context.Context, in io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r := in
	if f, ok := in.(*os.File); ok && isTerminal(f) {
		cr, err := cancelreader.NewReader(f)
		if err == nil {
			defer cr.Close()
			stop := context.AfterFunc(ctx, func() { cr.Cancel() })
			defer stop()
			r = cr
		}
	}
	var b strings.Builder
	one := make([]byte, 1)
	for {
		n, err := r.Read(one)
		if n == 1 {
			if one[0] == '\n' {
				return strings.TrimSpace(b.String()), nil
			}
			b.WriteByte(one[0])
		}
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, cancelreader.ErrCanceled) {
				return "", context.Canceled
			}
			if errors.Is(err, io.EOF) && b.Len() > 0 {
				return strings.TrimSpace(b.String()), nil
			}
			return "", fmt.Errorf("read the answer: %w", err)
		}
	}
}

// ask prints a question and reads the answer.
func (g *guide) ask(q string) (string, error) {
	fmt.Fprint(g.out, q)
	return readLine(g.ctx, g.in)
}

// yesNo asks a yes-or-no question. An empty answer is def.
func (g *guide) yesNo(q string, def bool) (bool, error) {
	opts := " [y/N] "
	if def {
		opts = " [Y/n] "
	}
	a, err := g.ask(q + opts)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(a) {
	case "":
		return def, nil
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

// pick asks for a number from 1 to n, again until it gets one. With text,
// any other answer that is not empty comes back as it is.
func (g *guide) pick(q string, n int, text bool) (int, string, error) {
	for {
		a, err := g.ask(q)
		if err != nil {
			return 0, "", err
		}
		if i, err := strconv.Atoi(a); err == nil && i >= 1 && i <= n {
			return i, "", nil
		}
		if text && a != "" {
			return 0, a, nil
		}
		fmt.Fprintf(g.out, "Type a number from 1 to %d.\n", n)
	}
}

// heading starts a numbered part of the setup.
func (g *guide) heading(title string) {
	g.part++
	fmt.Fprintf(g.out, "\n%d. %s\n\n", g.part, title)
}

// steps prints things to do, with each command indented under its words.
func (g *guide) steps(ss []onboard.Step) {
	for _, s := range ss {
		if s.Say != "" {
			fmt.Fprintf(g.out, "  %s\n", s.Say)
		}
		if s.Run != "" {
			fmt.Fprintf(g.out, "      %s\n", s.Run)
		}
	}
}
