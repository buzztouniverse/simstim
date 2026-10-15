package client

import (
	"os"
	"testing"
)

func TestRemotePath(t *testing.T) {
	h, _ := os.UserHomeDir()
	for in, want := range map[string]string{h: "~", h + "/src/app": "~/src/app", "/srv/app": "/srv/app", "~/x": "~/x", h + "other/x": h + "other/x"} {
		if got := RemotePath(in); got != want {
			t.Errorf("RemotePath(%q) = %q, want %q", in, got, want)
		}
	}
}
