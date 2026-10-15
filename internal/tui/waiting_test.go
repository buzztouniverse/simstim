package tui

import (
	"strings"
	"testing"

	"github.com/iAmLakshya/simstim/internal/api"
)

// w goes to the session that needs you, from anywhere, and says why.
func TestNextWaitingFromTheHosts(t *testing.T) {
	hs := home(t, 80, 24)
	hs.key("w")
	if hs.m.screen != scrSessions || hs.m.cur != "atlas-dev" || hs.m.sel != (sessKey{"atlas-dev", "s4"}) {
		t.Fatalf("screen %v, host %q, selected %v", hs.m.screen, hs.m.cur, hs.m.sel)
	}
	texts := toastTexts(hs.m)
	if len(texts) == 0 || !strings.HasPrefix(texts[len(texts)-1], "Auth-refactor on atlas-dev needs you") {
		t.Fatalf("toasts %q", texts)
	}
}

// Pressing w again goes to the next one, on another host, and back round.
func TestNextWaitingCycles(t *testing.T) {
	hs := home(t, 80, 24)
	gcp := hs.m.hostByName("gcp-dev")
	gcp.sessions[0].State = api.StateNeedsInput
	want := []sessKey{{"atlas-dev", "s4"}, {"gcp-dev", gcp.sessions[0].ID}, {"atlas-dev", "s4"}}
	for i, w := range want {
		hs.key("w")
		if hs.m.sel != w {
			t.Fatalf("press %d: selected %v, want %v", i+1, hs.m.sel, w)
		}
	}
}

func TestNextWaitingWithNoneWaiting(t *testing.T) {
	hs := home(t, 80, 24)
	for _, h := range hs.m.hosts {
		for i := range h.sessions {
			h.sessions[i].State = api.StateIdle
		}
	}
	hs.key("w")
	hasToast(t, hs.m, "No session needs you.")
	if hs.m.screen != scrHosts {
		t.Error("left the hosts with nothing to show")
	}
}
