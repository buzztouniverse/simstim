package client

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunProvider(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the test scripts are POSIX shell; on Windows a provider is a .cmd or .exe")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "provider.sh")
	// The script echoes its action and the job it received on stdin.
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '{\"action\":\"%s\",\"job\":' \"$1\"; cat; printf '}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	h := Host{Name: "box", SSH: "box.example", Labels: map[string]string{"provider": "test"}, Provider: &Provider{Script: script}}
	out, err := RunProvider(context.Background(), h, "status")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Action string `json:"action"`
		Job    struct {
			Action string         `json:"action"`
			Host   map[string]any `json:"host"`
		} `json:"job"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("bad output %q: %v", out, err)
	}
	if got.Action != "status" || got.Job.Action != "status" || got.Job.Host["name"] != "box" {
		t.Fatalf("got %+v", got)
	}

	failing := filepath.Join(dir, "fail.sh")
	_ = os.WriteFile(failing, []byte("#!/bin/sh\necho 'no credentials' >&2\nexit 3\n"), 0o755)
	h.Provider.Script = failing
	_, err = RunProvider(context.Background(), h, "start")
	if err == nil || !strings.Contains(err.Error(), "no credentials") {
		t.Fatalf("want the script's stderr in the error, got %v", err)
	}
}
