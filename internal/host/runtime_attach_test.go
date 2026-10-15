package host_test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/iAmLakshya/simstim/internal/host"
	"github.com/iAmLakshya/simstim/internal/host/hosttest"
	"github.com/iAmLakshya/simstim/internal/proto"
)

// A client gets all of the agent's output before it hears that the agent
// ended: the last words of a program are often why it ended. The output is
// written just before the exit, so the end races with the reading of it;
// many sessions at once make the end likelier to win.
func TestLastOutputComesBeforeTheEnd(t *testing.T) {
	hosttest.New(t)
	const n = 16
	ids := make([]string, n)
	var wg sync.WaitGroup
	for i := range ids {
		wg.Go(func() {
			s, err := host.New(host.NewArgs{Name: fmt.Sprint("last", i), Cwd: "~", Argv: []string{"/bin/sh", "-c",
				"stty raw -echo; printf 'START'; head -c 1 >/dev/null; printf 'LAST-OUTPUT'; exit 3"}})
			if err != nil {
				t.Error(err)
				return
			}
			ids[i] = s.ID
		})
	}
	wg.Wait()
	if t.Failed() {
		t.FailNow()
	}
	bridges := make([]*rawBridge, n)
	for i, id := range ids {
		waitOutput(t, id, "START")
		bridges[i] = newRawBridge(t)
		bridges[i].attach(id, -1, 80, 24)
	}
	for _, b := range bridges {
		b.send(proto.TInput, []byte("x"))
	}
	for i, b := range bridges {
		var got []byte
		for {
			f := b.next(proto.TOutput, proto.TEnded)
			if f.Type == proto.TEnded {
				break
			}
			_, data, _ := proto.DecodeOutput(f.Payload)
			got = append(got, data...)
		}
		if !bytes.Contains(got, []byte("LAST-OUTPUT")) {
			t.Errorf("session %d: the end came after only %q", i, got)
		}
	}
}
