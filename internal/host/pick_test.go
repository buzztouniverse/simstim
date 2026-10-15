package host

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/iAmLakshya/simstim/internal/proto"
	"github.com/iAmLakshya/simstim/internal/version"
)

var pickSessions = []SessionInfo{
	{ID: "aaaa1111", Name: "auth-fix", Agent: "claude", State: "needs_input", Alive: true},
	{ID: "bbbb2222", Name: "api-tests-with-a-very-long-name", Agent: "codex", State: "working", Alive: true},
	{ID: "cccc3333", Name: "notes", Agent: "shell", State: "idle", Alive: true},
	{ID: "dddd4444", Name: "notes", Agent: "shell", State: "idle", Alive: true},
}

func TestParseChoice(t *testing.T) {
	cases := []struct {
		in   string
		i    int
		quit bool
		err  bool
	}{
		{"1\n", 0, false, false},
		{" 3 \r\n", 2, false, false},
		{"q\n", 0, true, false},
		{"QUIT", 0, true, false},
		{"0", 0, false, true},
		{"5", 0, false, true},
		{"", 0, false, true},
		{"auth-fix", 0, false, false},
		{"bbbb2222", 1, false, false},
		{"notes", 0, false, true}, // two sessions have this name
		{"nope", 0, false, true},
	}
	for _, c := range cases {
		i, quit, err := parseChoice(c.in, pickSessions)
		if i != c.i || quit != c.quit || (err != nil) != c.err {
			t.Errorf("parseChoice(%q) = %d, %v, %v", c.in, i, quit, err)
		}
	}
}

func TestMenuFitsASmallTerminal(t *testing.T) {
	now := time.Now()
	list := append([]SessionInfo(nil), pickSessions...)
	list[0].CreatedAt = now.Add(-3 * time.Minute)
	list[1].CreatedAt = now.Add(-5 * time.Hour)
	list[2].CreatedAt = now.Add(-9 * 24 * time.Hour)
	out := menu("box", list, now, 40)
	if strings.ContainsRune(out, 0x1b) {
		t.Error("the menu has escape sequences")
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if n := utf8.RuneCountInString(line); n > 40 {
			t.Errorf("%q is %d cells wide", line, n)
		}
	}
	for _, want := range []string{"Sessions on box", " 1  auth-fix", "needs input", "3m", "5h", "9d", " 4  notes"} {
		if !strings.Contains(out, want) {
			t.Errorf("the menu lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "claude") {
		t.Error("a 40-column menu has room for the agent")
	}
	if wide := menu("box", list, now, 80); !strings.Contains(wide, " 1  auth-fix") || !strings.Contains(wide, "claude") {
		t.Errorf("an 80-column menu lacks the agent:\n%s", wide)
	}
	if p := pickable(append(list, SessionInfo{ID: "x", Alive: false})); len(p) != 4 {
		t.Errorf("pickable kept %d sessions", len(p))
	}
}

func TestReadChoice(t *testing.T) {
	var out bytes.Buffer
	r := bufio.NewReader(strings.NewReader("9\nhello\n2\n"))
	i, ok := readChoice(r, &out, pickSessions)
	if !ok || i != 1 || strings.Count(out.String(), "Pick 1-4") != 3 || !strings.Contains(out.String(), "no session hello") {
		t.Errorf("readChoice = %d, %v; out %q", i, ok, out.String())
	}
	if _, ok := readChoice(bufio.NewReader(strings.NewReader("")), &out, pickSessions); ok {
		t.Error("end of input did not quit")
	}
}

// syncBuffer is a bytes.Buffer safe to read while the pump writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// fakeHolder answers an attach on conn, sends frames, then reports every frame
// it gets on the returned channel.
func fakeHolder(t *testing.T, conn net.Conn, attached proto.Attached, send ...proto.Frame) (<-chan proto.Frame, <-chan proto.Attach) {
	t.Helper()
	got := make(chan proto.Frame, 16)
	att := make(chan proto.Attach, 1)
	go func() {
		defer close(got)
		r, w := bufio.NewReader(conn), proto.NewWriter(conn)
		if f, err := proto.Read(r); err != nil || f.Type != proto.THello {
			return
		}
		_ = w.WriteJSON(proto.THello, proto.Hello{Magic: proto.Magic, Protocol: version.Protocol, Role: "holder"})
		f, err := proto.Read(r)
		if err != nil || f.Type != proto.TAttach {
			return
		}
		var a proto.Attach
		_ = json.Unmarshal(f.Payload, &a)
		att <- a
		_ = w.WriteJSON(proto.TAttached, attached)
		for _, f := range send {
			_ = w.Write(f.Type, f.Payload)
		}
		for {
			f, err := proto.Read(r)
			if err != nil {
				return
			}
			got <- f
		}
	}()
	return got, att
}

func expectFrame(t *testing.T, got <-chan proto.Frame, typ proto.Type) proto.Frame {
	t.Helper()
	select {
	case f := <-got:
		if f.Type != typ {
			t.Fatalf("got frame %x, want %x", f.Type, typ)
		}
		return f
	case <-time.After(5 * time.Second):
		t.Fatalf("no frame %x", typ)
	}
	return proto.Frame{}
}

func TestPickPump(t *testing.T) {
	holder, conn := net.Pipe()
	defer holder.Close()
	defer conn.Close()
	history := "old\x1b[c"                        // a query in history must not reach the terminal
	live := "new\x1b[c\x1b[?1049h\x1b]8;;x\x1b\\" // live bytes pass byte for byte
	got, att := fakeHolder(t, holder, proto.Attached{Session: "s1", Mode: "full", From: 0, Live: int64(len(history))},
		proto.Frame{Type: proto.TOutput, Payload: proto.EncodeOutput(0, []byte(history))},
		proto.Frame{Type: proto.TOutput, Payload: proto.EncodeOutput(int64(len(history)), []byte(live))})

	out := &syncBuffer{}
	winch := make(chan os.Signal, 1)
	size := []int{40, 20}
	var mu sync.Mutex
	p := &pickPump{conn: conn, out: out, winch: winch, size: func() (int, int) {
		mu.Lock()
		defer mu.Unlock()
		return size[0], size[1]
	}}
	input := make(chan []byte, 8)
	type result struct {
		end pickEnd
		err error
	}
	done := make(chan result, 1)
	go func() {
		end, err := p.run("s1", input)
		done <- result{end, err}
	}()

	if a := <-att; a.Session != "s1" || a.Cols != 40 || a.Rows != 20 || a.LastOffset != -1 {
		t.Errorf("attach = %+v", a)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), live) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	input <- []byte("ls -l\r\x1b[A")
	if f := expectFrame(t, got, proto.TInput); string(f.Payload) != "ls -l\r\x1b[A" {
		t.Errorf("input = %q", f.Payload)
	}
	mu.Lock()
	size = []int{60, 30}
	mu.Unlock()
	winch <- os.Interrupt
	var rz proto.Resize
	if f := expectFrame(t, got, proto.TResize); json.Unmarshal(f.Payload, &rz) != nil || rz.Cols != 60 || rz.Rows != 30 {
		t.Errorf("resize = %s", f.Payload)
	}
	input <- []byte("\x1c")
	r := <-done
	if r.err != nil || r.end.ended != nil {
		t.Fatalf("run = %+v, %v", r.end, r.err)
	}
	s := out.String()
	if !strings.Contains(s, live) {
		t.Errorf("live output was changed: %q", s)
	}
	if strings.Contains(s, history) || !strings.Contains(s, "old") {
		t.Errorf("history kept its query: %q", s)
	}
	if !strings.HasSuffix(s, string(p.tracker.M.ResetSequence())) || !strings.Contains(string(p.tracker.M.ResetSequence()), "\x1b[?1049l") {
		t.Errorf("the session's modes were not undone: %q", s)
	}
	if strings.Contains(s, "\x1bc") {
		t.Error("RIS was sent")
	}
	select {
	case f, ok := <-got:
		if ok && f.Type == proto.TInput {
			t.Errorf("the detach key reached the session: %q", f.Payload)
		}
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPickPumpEnded(t *testing.T) {
	holder, conn := net.Pipe()
	defer holder.Close()
	defer conn.Close()
	ended, _ := json.Marshal(proto.Ended{ExitCode: 3})
	fakeHolder(t, holder, proto.Attached{Session: "s1", Mode: "full", Exited: true}, proto.Frame{Type: proto.TEnded, Payload: ended})
	p := &pickPump{conn: conn, out: &syncBuffer{}, size: func() (int, int) { return 80, 24 }}
	end, err := p.run("s1", make(chan []byte))
	if err != nil || end.ended == nil || end.ended.ExitCode != 3 {
		t.Errorf("run = %+v, %v", end, err)
	}

	// A holder that goes away ends the attach with an error.
	holder2, conn2 := net.Pipe()
	defer conn2.Close()
	go func() {
		r, w := bufio.NewReader(holder2), proto.NewWriter(holder2)
		_, _ = proto.Read(r)
		_ = w.WriteJSON(proto.THello, proto.Hello{Magic: proto.Magic, Protocol: version.Protocol, Role: "holder"})
		_, _ = proto.Read(r)
		holder2.Close()
	}()
	p = &pickPump{conn: conn2, out: &syncBuffer{}, size: func() (int, int) { return 80, 24 }}
	if _, err := p.run("s1", make(chan []byte)); err == nil {
		t.Error("a vanished holder did not end the attach")
	}
}

func TestReadInputSendsBufferedKeysFirst(t *testing.T) {
	input := make(chan []byte, 4)
	stop, done := make(chan struct{}), make(chan struct{})
	go readInput([]byte("ab"), strings.NewReader("cd"), input, stop, done)
	var all []byte
	for b := range input {
		all = append(all, b...)
	}
	<-done
	if string(all) != "abcd" {
		t.Errorf("input = %q", all)
	}
}
