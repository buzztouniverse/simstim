package alerts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/brand"
)

// got is one request a test server received.
type got struct {
	header http.Header
	body   []byte
	path   string
}

// server answers with the given codes in turn (200 when they run out) and
// records each request.
func server(t *testing.T, codes ...int) (*httptest.Server, func() []got) {
	t.Helper()
	var mu sync.Mutex
	var reqs []got
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		n := len(reqs)
		reqs = append(reqs, got{header: r.Header.Clone(), body: b, path: r.URL.Path})
		mu.Unlock()
		code := 200
		if n < len(codes) {
			code = codes[n]
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []got {
		mu.Lock()
		defer mu.Unlock()
		return append([]got(nil), reqs...)
	}
}

func testEvent(kind string) Event {
	code := 2
	ev := Event{Event: kind, Session: Session{ID: "abc123", Name: "auth-fix", Agent: "claude", Dir: "/home/u/src"}, at: time.Now()}
	if kind == Failed {
		ev.ExitCode = &code
	}
	return prepare(ev, "box", &Settings{Detail: true})
}

func TestNtfy(t *testing.T) {
	srv, reqs := server(t)
	s := newSink(api.AlertDestination{Type: "ntfy", URL: srv.URL + "/my-topic", Token: "tk_test"}, sinkEnv{client: NewHTTPClient()})
	ev := testEvent(NeedsInput)
	if _, err := s.send(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	r := reqs()[0]
	if r.path != "/my-topic" || string(r.body) != "auth-fix on box needs you" {
		t.Errorf("request = %s %q", r.path, r.body)
	}
	if r.header.Get("Authorization") != "Bearer tk_test" || r.header.Get("Priority") != "high" || r.header.Get("Tags") != "bell" {
		t.Errorf("headers = %v", r.header)
	}
	if title := r.header.Get("Title"); !strings.Contains(title, "box") {
		t.Errorf("title = %q", title)
	}
}

func TestNtfyTokenFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no Unix modes")
	}
	srv, reqs := server(t)
	tok := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tok, []byte("tk_test_file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := api.AlertDestination{Type: "ntfy", URL: srv.URL + "/t", TokenFile: tok, Priority: "2"}
	if _, err := newSink(d, sinkEnv{client: NewHTTPClient()}).send(context.Background(), testEvent(Failed)); err != nil {
		t.Fatal(err)
	}
	r := reqs()[0]
	if r.header.Get("Authorization") != "Bearer tk_test_file" || r.header.Get("Priority") != "2" {
		t.Errorf("headers = %v", r.header)
	}
	if err := os.Chmod(tok, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := newSink(d, sinkEnv{client: NewHTTPClient()}).send(context.Background(), testEvent(Failed)); err == nil || !strings.Contains(err.Error(), "other users") {
		t.Errorf("a readable token file was used: %v", err)
	}
	if len(reqs()) != 1 {
		t.Error("the request went out without its token")
	}
}

func TestWebhookFormats(t *testing.T) {
	srv, reqs := server(t)
	ev := testEvent(Failed)
	for _, format := range []string{"", "json", "slack", "discord"} {
		d := api.AlertDestination{Type: "webhook", URL: srv.URL + "/hook", Format: format, Headers: map[string]string{"X-Key": "k"}}
		if _, err := newSink(d, sinkEnv{client: NewHTTPClient()}).send(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}
	all := reqs()
	var e Event
	if err := json.Unmarshal(all[0].body, &e); err != nil || e.V != 1 || e.Event != Failed || *e.ExitCode != 2 || e.Session.Name != "auth-fix" {
		t.Errorf("json body = %s (%v)", all[0].body, err)
	}
	if string(all[1].body) != string(all[0].body) {
		t.Error("json is not the default format")
	}
	var slack map[string]string
	if json.Unmarshal(all[2].body, &slack) != nil || slack["text"] != "auth-fix on box failed (exit 2)" {
		t.Errorf("slack body = %s", all[2].body)
	}
	var discord map[string]string
	if json.Unmarshal(all[3].body, &discord) != nil || discord["content"] != "auth-fix on box failed (exit 2)" {
		t.Errorf("discord body = %s", all[3].body)
	}
	for _, r := range all {
		if r.header.Get("Content-Type") != "application/json" || r.header.Get("X-Key") != "k" {
			t.Errorf("headers = %v", r.header)
		}
	}
}

func TestRetry(t *testing.T) {
	srv, reqs := server(t, 503, 200)
	s := newSink(api.AlertDestination{Type: "webhook", URL: srv.URL}, sinkEnv{client: NewHTTPClient()})
	if _, err := deliver(context.Background(), s, testEvent(Failed), time.Millisecond); err != nil {
		t.Errorf("a 503 then 200 failed: %v", err)
	}
	if n := len(reqs()); n != 2 {
		t.Errorf("%d requests, want 2", n)
	}

	srv, reqs = server(t, 400, 200)
	s = newSink(api.AlertDestination{Type: "webhook", URL: srv.URL}, sinkEnv{client: NewHTTPClient()})
	_, err := deliver(context.Background(), s, testEvent(Failed), time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") || !strings.Contains(err.Error(), "nope") {
		t.Errorf("a 400 = %v", err)
	}
	if n := len(reqs()); n != 1 {
		t.Errorf("a 400 was retried: %d requests", n)
	}

	srv, reqs = server(t, 500, 502)
	s = newSink(api.AlertDestination{Type: "webhook", URL: srv.URL}, sinkEnv{client: NewHTTPClient()})
	if _, err := deliver(context.Background(), s, testEvent(Failed), time.Millisecond); err == nil {
		t.Error("two 5xx answers counted as sent")
	}
	if n := len(reqs()); n != 2 {
		t.Errorf("%d requests, want one try and one retry", n)
	}
}

func TestNetworkErrorHidesTheAddress(t *testing.T) {
	srv, _ := server(t)
	u := srv.URL + "/secret-topic-xyz"
	srv.Close()
	s := newSink(api.AlertDestination{Type: "ntfy", URL: u}, sinkEnv{client: NewHTTPClient()})
	_, err := deliver(context.Background(), s, testEvent(Failed), time.Millisecond)
	if err == nil || !retryable(err) {
		t.Fatalf("err = %v, want a retryable network error", err)
	}
	if strings.Contains(err.Error(), "secret-topic-xyz") {
		t.Errorf("the error shows the address: %v", err)
	}
}

func TestNoRedirectToAnotherHost(t *testing.T) {
	other, otherReqs := server(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/stolen", http.StatusTemporaryRedirect)
	}))
	defer srv.Close()
	// Both servers are on 127.0.0.1, so tell them apart by port: Host has it.
	s := newSink(api.AlertDestination{Type: "webhook", URL: srv.URL}, sinkEnv{client: NewHTTPClient()})
	_, err := s.send(context.Background(), testEvent(Failed))
	if err == nil || !strings.Contains(err.Error(), "307") {
		t.Errorf("err = %v, want the redirect refused", err)
	}
	if len(otherReqs()) != 0 {
		t.Error("the redirect was followed to another host")
	}
}

func TestCommandSink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("command destinations run on Linux hosts")
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "out.json")
	name := filepath.Join(dir, "name.txt")
	run := `cat > "` + out + `"; printf '%s|%s|%s' "$` + brand.Env("SESSION_NAME") + `" "$` + brand.Env("EVENT") + `" "$` + brand.Env("EXIT_CODE") + `" > "` + name + `"; echo done`
	s := newSink(api.AlertDestination{Type: "command", Run: run}, sinkEnv{env: os.Environ(), dir: dir})

	// Values that a shell would run, if they were ever put into a command line.
	evil := "$(touch " + filepath.Join(dir, "pwned1") + ")`touch " + filepath.Join(dir, "pwned2") + "`;touch " + filepath.Join(dir, "pwned3")
	ev := testEvent(Failed)
	ev.Session.Name = evil
	ev.Detail = evil
	ev.Summary = evil
	text, err := s.send(context.Background(), ev)
	if err != nil || text != "done" {
		t.Fatalf("send = %q, %v", text, err)
	}
	for _, p := range []string{"pwned1", "pwned2", "pwned3"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err == nil {
			t.Errorf("an event value was run as a command (%s exists)", p)
		}
	}
	b, _ := os.ReadFile(name)
	if string(b) != evil+"|failed|2" {
		t.Errorf("environment = %q", b)
	}
	var back Event
	raw, _ := os.ReadFile(out)
	if err := json.Unmarshal(raw, &back); err != nil || back.Session.Name != evil || back.Event != Failed {
		t.Errorf("stdin = %s (%v)", raw, err)
	}
}

func TestCommandSinkFailsAndTimesOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("command destinations run on Linux hosts")
	}
	s := newSink(api.AlertDestination{Type: "command", Run: "echo oops >&2; exit 3"}, sinkEnv{env: os.Environ()})
	text, err := s.send(context.Background(), testEvent(Failed))
	if err == nil || !strings.Contains(err.Error(), "exit status 3") || text != "oops" {
		t.Errorf("send = %q, %v", text, err)
	}
	if retryable(err) {
		t.Error("a failed command is retried")
	}

	// A command that leaves a child holding its output still succeeds.
	s = newSink(api.AlertDestination{Type: "command", Run: "sleep 3 & echo sent"}, sinkEnv{env: os.Environ()})
	if text, err := s.send(context.Background(), testEvent(Failed)); err != nil || text != "sent" {
		t.Errorf("send = %q, %v", text, err)
	}

	old := commandTimeout
	commandTimeout = 200 * time.Millisecond
	defer func() { commandTimeout = old }()
	start := time.Now()
	s = newSink(api.AlertDestination{Type: "command", Run: "sleep 30 & sleep 30"}, sinkEnv{env: os.Environ()})
	_, err = s.send(context.Background(), testEvent(Failed))
	if err == nil || !strings.Contains(err.Error(), "longer than") {
		t.Errorf("err = %v, want a timeout", err)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("the timeout took %s", took)
	}
}

func TestCommandOutputIsCapped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("command destinations run on Linux hosts")
	}
	s := newSink(api.AlertDestination{Type: "command", Run: "head -c 100000 /dev/zero | tr '\\0' x"}, sinkEnv{env: os.Environ()})
	text, err := s.send(context.Background(), testEvent(Failed))
	if err != nil {
		t.Fatal(err)
	}
	if len(text) > maxOutput+len("…") || !strings.HasSuffix(text, "…") {
		t.Errorf("output is %d bytes", len(text))
	}
}
