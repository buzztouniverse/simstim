package cli

import (
	"strings"
	"testing"

	"github.com/iAmLakshya/simstim/internal/client"
)

func TestScrollbackAdvice(t *testing.T) {
	got := scrollbackAdvice([]client.Session{
		{Name: "a", Agent: "claude", Alive: true, FullScreen: true},
		{Name: "b", Agent: "opencode", Alive: true, FullScreen: true},
		{Name: "c", Agent: "codex", Alive: true},                     // inline: no advice
		{Name: "d", Agent: "claude", Alive: false, FullScreen: true}, // ended: no advice
	})
	if len(got) != 2 {
		t.Fatalf("advice %q", got)
	}
	if !strings.Contains(got[0], "--inline") || !strings.Contains(got[0], "Claude Code") {
		t.Errorf("claude advice: %q", got[0])
	}
	if !strings.Contains(got[1], "no inline mode") {
		t.Errorf("opencode advice: %q", got[1])
	}
}
