package e2b

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/iAmLakshya/simstim/internal/brand"
)

const (
	testKey   = "e2b_fake_key"
	testID    = "i1a2b3c4d5e6f7g8h9j0k"
	testToken = "fake-envd-token"
)

// fixtures reads testdata/api.json, with {{brand}} as the product name.
func fixtures(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "api.json"))
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.ReplaceAll(b, []byte("{{brand}}"), []byte(brand.Name))
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

type fixture struct {
	Method   string          `json:"method"`
	Path     string          `json:"path"`
	Request  json.RawMessage `json:"request"`
	Status   int             `json:"status"`
	Response json.RawMessage `json:"response"`
}

func entry(t *testing.T, all map[string]json.RawMessage, name string) fixture {
	t.Helper()
	var f fixture
	if err := json.Unmarshal(all[name], &f); err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return f
}

type call struct {
	Method, Path string
	Body         map[string]any
}

// fakeAPI answers the E2B REST API from the recorded shapes, and keeps the
// sandbox's state.
type fakeAPI struct {
	t     *testing.T
	all   map[string]json.RawMessage
	mu    sync.Mutex
	state string // "", running, paused, gone
	calls []call
	tmpl  string // the templates fixture to serve
}

func newFakeAPI(t *testing.T) (*fakeAPI, *httptest.Server) {
	f := &fakeAPI{t: t, all: fixtures(t), tmpl: "templates"}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fakeAPI) record(r *http.Request) {
	var body map[string]any
	if b, _ := io.ReadAll(r.Body); len(b) > 0 {
		if err := json.Unmarshal(b, &body); err != nil {
			f.t.Errorf("%s %s: the body is not JSON: %q", r.Method, r.URL.Path, b)
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, call{r.Method, r.URL.Path, body})
	f.mu.Unlock()
}

func (f *fakeAPI) Calls() []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]call(nil), f.calls...)
}

func (f *fakeAPI) answer(w http.ResponseWriter, name string, status int) {
	e := entry(f.t, f.all, name)
	if status == 0 {
		status = e.Status
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if len(e.Response) > 0 && status != http.StatusNoContent {
		_, _ = w.Write(e.Response)
	}
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.record(r)
	if r.Header.Get("X-API-Key") != testKey {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":401,"message":"Invalid API key"}`))
		return
	}
	f.mu.Lock()
	state := f.state
	f.mu.Unlock()
	set := func(s string) { f.mu.Lock(); f.state = s; f.mu.Unlock() }
	sb := "/sandboxes/" + testID
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
		set("running")
		f.answer(w, "create", 0)
	case r.Method == http.MethodGet && r.URL.Path == "/templates":
		f.answer(w, f.tmpl, 0)
	case state == "gone" || state == "":
		f.answer(w, "not_found", 0)
	case r.Method == http.MethodGet && r.URL.Path == sb:
		f.answer(w, "get_"+state, 0)
	case r.Method == http.MethodPost && r.URL.Path == "/v2"+sb+"/connect":
		set("running")
		if state == "running" {
			f.answer(w, "connect", http.StatusOK)
		} else {
			f.answer(w, "connect", 0)
		}
	case r.Method == http.MethodPost && r.URL.Path == sb+"/pause":
		set("paused")
		f.answer(w, "pause", 0)
	case r.Method == http.MethodPost && r.URL.Path == sb+"/timeout":
		f.answer(w, "timeout", 0)
	case r.Method == http.MethodDelete && r.URL.Path == sb:
		set("gone")
		f.answer(w, "kill", 0)
	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusTeapot)
	}
}

// fakeEnvd runs commands on this machine behind envd's process service, and
// checks every request's headers.
type fakeEnvd struct {
	t     *testing.T
	mu    sync.Mutex
	procs map[uint32]*exec.Cmd
	stdin map[uint32]io.WriteCloser
	paths []string
	tags  []string
	sigs  []string
	// drop, when set, makes Start end the stream right after the start
	// event, as a dropped connection would.
	drop bool
	// bridge, when set, serves commands that run the helper's bridge in
	// this process, instead of running them.
	bridge  func(in io.Reader, out io.Writer, remote string) error
	bridges []string
}

// serveBridge runs the bridge in this process, as if it were a command in
// the sandbox, with a made-up process ID.
func (f *fakeEnvd) serveBridge(w http.ResponseWriter, remote string, bridge func(io.Reader, io.Writer, string) error) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	f.mu.Lock()
	pid := uint32(1_000_000 + len(f.bridges))
	f.stdin[pid] = inW
	f.bridges = append(f.bridges, remote)
	f.mu.Unlock()
	w.Header().Set("Content-Type", contentStream)
	w.WriteHeader(http.StatusOK)
	fw := &flushWriter{w: w}
	fw.send(0, map[string]any{"event": map[string]any{"start": map[string]any{"pid": pid}}})
	go func() {
		_ = bridge(inR, outW, remote)
		_ = outW.Close()
	}()
	buf := make([]byte, 32<<10)
	for {
		n, err := outR.Read(buf)
		if n > 0 {
			fw.send(0, map[string]any{"event": map[string]any{"data": map[string]any{"stdout": buf[:n]}}})
		}
		if err != nil {
			break
		}
	}
	fw.send(0, map[string]any{"event": map[string]any{"end": map[string]any{"exited": true}}})
	fw.send(flagEndStream, map[string]any{})
}

func newFakeEnvd(t *testing.T) (*fakeEnvd, *httptest.Server) {
	f := &fakeEnvd{t: t, procs: map[uint32]*exec.Cmd{}, stdin: map[uint32]io.WriteCloser{}}
	srv := httptest.NewServer(f)
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, c := range f.procs {
			_ = c.Process.Kill()
		}
	})
	return f, srv
}

func (f *fakeEnvd) Paths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.paths...)
}

func (f *fakeEnvd) check(r *http.Request, stream bool) bool {
	f.mu.Lock()
	f.paths = append(f.paths, strings.TrimPrefix(r.URL.Path, "/process.Process/"))
	f.mu.Unlock()
	want := contentUnary
	if stream {
		want = contentStream
	}
	user, _, ok := r.BasicAuth()
	switch {
	case !strings.HasPrefix(r.Header.Get("X-Access-Token"), testToken): // a resume may hand out a new one
		f.t.Errorf("%s: X-Access-Token = %q", r.URL.Path, r.Header.Get("X-Access-Token"))
	case r.Header.Get("Content-Type") != want:
		f.t.Errorf("%s: Content-Type = %q, want %q", r.URL.Path, r.Header.Get("Content-Type"), want)
	case r.Header.Get("Connect-Timeout-Ms") != "":
		f.t.Errorf("%s: a Connect-Timeout-Ms header was sent; envd would kill the process when it ends", r.URL.Path)
	case !ok || user != "user":
		f.t.Errorf("%s: basic auth user = %q", r.URL.Path, user)
	case r.Header.Get("E2b-Sandbox-Id") != testID:
		f.t.Errorf("%s: E2b-Sandbox-Id = %q", r.URL.Path, r.Header.Get("E2b-Sandbox-Id"))
	default:
		return true
	}
	return false
}

func (f *fakeEnvd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch strings.TrimPrefix(r.URL.Path, "/process.Process/") {
	case "Start":
		if f.check(r, true) {
			f.start(w, r)
		}
	case "StreamInput":
		if f.check(r, true) {
			f.streamInput(w, r)
		}
	case "SendInput", "CloseStdin", "SendSignal":
		if f.check(r, false) {
			f.unary(w, r)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

type flushWriter struct {
	mu sync.Mutex
	w  http.ResponseWriter
}

func (fw *flushWriter) send(flags byte, v any) {
	b, _ := json.Marshal(v)
	fw.mu.Lock()
	defer fw.mu.Unlock()
	_ = writeEnvelope(fw.w, flags, b)
	fw.w.(http.Flusher).Flush()
}

func (f *fakeEnvd) start(w http.ResponseWriter, r *http.Request) {
	_, msg, err := readEnvelope(r.Body)
	if err != nil {
		f.t.Errorf("Start: %v", err)
		return
	}
	var req startRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		f.t.Errorf("Start: %v", err)
		return
	}
	if req.Process.Cmd != "/bin/sh" || len(req.Process.Args) != 2 || req.Process.Args[0] != "-c" || !req.Stdin || !strings.HasPrefix(req.Tag, brand.Name+"-") {
		f.t.Errorf("Start request = %+v", req)
	}
	if r.Header.Get("Keepalive-Ping-Interval") == "" {
		f.t.Error("Start: no Keepalive-Ping-Interval")
	}
	f.mu.Lock()
	bridge := f.bridge
	f.mu.Unlock()
	if bridge != nil && strings.Contains(req.Process.Args[1], " bridge") {
		f.serveBridge(w, req.Process.Args[1], bridge)
		return
	}
	cmd := exec.Command("/bin/sh", "-c", req.Process.Args[1])
	in, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	errp, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		f.t.Errorf("Start: %v", err)
		return
	}
	pid := uint32(cmd.Process.Pid)
	f.mu.Lock()
	f.procs[pid], f.stdin[pid] = cmd, in
	f.tags = append(f.tags, req.Tag)
	drop := f.drop
	f.mu.Unlock()
	w.Header().Set("Content-Type", contentStream)
	w.WriteHeader(http.StatusOK)
	fw := &flushWriter{w: w}
	fw.send(0, map[string]any{"event": map[string]any{"start": map[string]any{"pid": pid}}})
	fw.send(0, map[string]any{"event": map[string]any{"keepalive": map[string]any{}}})
	if drop {
		return // the handler returns: the stream ends with no end event
	}
	var wg sync.WaitGroup
	pipe := func(rd io.Reader, field string) {
		defer wg.Done()
		br := bufio.NewReader(rd)
		buf := make([]byte, 32<<10)
		for {
			n, err := br.Read(buf)
			if n > 0 {
				fw.send(0, map[string]any{"event": map[string]any{"data": map[string]any{field: buf[:n]}}})
			}
			if err != nil {
				return
			}
		}
	}
	wg.Add(2)
	go pipe(out, "stdout")
	go pipe(errp, "stderr")
	wg.Wait()
	end := map[string]any{"exited": true}
	if err := cmd.Wait(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				end = map[string]any{"exitCode": -1, "exited": false, "status": "signal: " + ws.Signal().String()}
			} else {
				end["exitCode"] = ee.ExitCode()
			}
		}
	}
	fw.send(0, map[string]any{"event": map[string]any{"end": end}})
	fw.send(flagEndStream, map[string]any{})
}

func (f *fakeEnvd) input(pid uint32, data []byte) error {
	f.mu.Lock()
	in := f.stdin[pid]
	f.mu.Unlock()
	if in == nil {
		return fmt.Errorf("process with pid %d not found", pid)
	}
	_, err := in.Write(data)
	return err
}

func (f *fakeEnvd) unary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Process selector     `json:"process"`
		Input   processInput `json:"input"`
		Signal  string       `json:"signal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.t.Errorf("%s: %v", r.URL.Path, err)
	}
	pid := req.Process.Pid
	var err error
	switch strings.TrimPrefix(r.URL.Path, "/process.Process/") {
	case "SendInput":
		err = f.input(pid, req.Input.Stdin)
	case "CloseStdin":
		f.mu.Lock()
		if in := f.stdin[pid]; in != nil {
			_ = in.Close()
		}
		f.mu.Unlock()
	case "SendSignal":
		f.mu.Lock()
		f.sigs = append(f.sigs, req.Signal)
		if c := f.procs[pid]; c != nil && req.Signal == "SIGNAL_SIGKILL" {
			_ = c.Process.Kill()
		}
		f.mu.Unlock()
	}
	w.Header().Set("Content-Type", contentUnary)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(rpcError{Code: "not_found", Message: err.Error()})
		return
	}
	_, _ = w.Write([]byte("{}"))
}

func (f *fakeEnvd) streamInput(w http.ResponseWriter, r *http.Request) {
	var pid uint32
	for {
		_, msg, err := readEnvelope(r.Body)
		if err != nil {
			break
		}
		var m streamInputRequest
		if err := json.Unmarshal(msg, &m); err != nil {
			f.t.Errorf("StreamInput: %v", err)
			return
		}
		switch {
		case m.Start != nil:
			pid = m.Start.Process.Pid
		case m.Data != nil:
			if err := f.input(pid, m.Data.Input.Stdin); err != nil {
				f.t.Errorf("StreamInput: %v", err)
			}
		}
	}
	w.Header().Set("Content-Type", contentStream)
	w.WriteHeader(http.StatusOK)
	fw := &flushWriter{w: w}
	fw.send(0, map[string]any{})
	fw.send(flagEndStream, map[string]any{})
}
