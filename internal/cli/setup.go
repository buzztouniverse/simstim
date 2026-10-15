package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/client"
)

// startSession starts a session with the user's answers about setup scripts
// on h. When the host finds a repository with a setup script and no answer,
// it asks on this terminal, keeps the answer, and starts again. Without a
// terminal it skips the script this time, and says so.
func startSession(ctx context.Context, conn *client.Conn, h client.Host, a client.NewArgs) (*client.Session, error) {
	if a.Setup == "" {
		a.SetupTrust = client.SetupTrust(h.Name)
	}
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	s, err := conn.NewSession(cctx, a)
	cancel()
	q, ok := client.SetupQuestionOf(err)
	if !ok {
		return s, err
	}
	if isTerminal(os.Stdin) {
		run := askSetup(os.Stdin, os.Stdout, h.Name, q)
		if err := client.RememberSetup(h.Name, q.Repo, run); err != nil {
			return nil, err
		}
		a.Setup = setupValue(run)
	} else {
		fmt.Fprintf(os.Stderr, "Skipping the setup script of %s: you have not said that it may run on %s. Pass --setup to run it.\n", q.Repo, h.Name)
		a.Setup = api.SetupSkip
	}
	cctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return conn.NewSession(cctx, a)
}

func setupValue(run bool) string {
	if run {
		return api.SetupRun
	}
	return api.SetupSkip
}

// askSetup asks whether a repository's setup script may run on host.
func askSetup(in io.Reader, w io.Writer, host string, q client.SetupQuestion) bool {
	fmt.Fprintf(w, "The repository %s on %s has a setup script:\n\n    %s\n\n", q.Repo, host, q.Script)
	fmt.Fprintf(w, "It runs in the session before the agent, as you, with the session's environment and\nsecrets. Say yes only for a repository you trust. The answer is kept for this\nrepository on %s; --setup and --no-setup override it.\n\n", host)
	return ask(bufio.NewReader(in), w, "Run it?")
}
