package tui

import (
	"strings"
	"testing"
)

// trustLab is the lab with a repository on atlas-dev whose setup script has
// no answer yet.
func trustLab(t *testing.T) *harness {
	t.Helper()
	be := labBackend()
	be.state["atlas-dev"].setupRepo = "/home/dev/src/app"
	hs := newHarness(t, be, 80, 24)
	hs.m.motion = false
	return hs
}

func TestQuickStartAsksAboutASetupScript(t *testing.T) {
	hs := trustLab(t)
	hs.key("n")
	hs.typeText("claude ~/src/app")
	hs.key("enter")
	ask, ok := hs.m.modal.(*setupAsk)
	if !ok {
		t.Fatalf("no question; modal %T, calls %q", hs.m.modal, hs.be.callList())
	}
	if ask.q.Repo != "/home/dev/src/app" || len(hs.be.attached) != 0 {
		t.Fatalf("question %+v, attached %d", ask.q, len(hs.be.attached))
	}
	out := plainScreen(hs.m)
	for _, want := range []string{"Run its setup script?", "The repository ~/src/app on atlas-dev has a setup script", "y run it", "n skip it"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	hs.key("y")
	hasCall(t, hs.be, "remember atlas-dev /home/dev/src/app run=true")
	hasCall(t, hs.be, "new atlas-dev setup=run")
	if len(hs.be.attached) != 1 || hs.m.prompt != nil || hs.m.modal != nil {
		t.Fatalf("after yes: attached %d, prompt %v, modal %T", len(hs.be.attached), hs.m.prompt, hs.m.modal)
	}

	// The next start in the repository asks nothing: the answer goes along.
	hs.key("n")
	hs.typeText("claude ~/src/app")
	hs.key("enter")
	if _, ok := hs.m.modal.(*setupAsk); ok || len(hs.be.attached) != 2 {
		t.Fatalf("asked again: modal %T, attached %d", hs.m.modal, len(hs.be.attached))
	}
}

func TestFormSetupScriptSkipAndCancel(t *testing.T) {
	hs := trustLab(t)
	hs.key("n", "ctrl+f")
	if _, ok := hs.m.modal.(*newSession); !ok {
		t.Fatalf("no form: %T", hs.m.modal)
	}
	hs.key("enter")
	if _, ok := hs.m.modal.(*setupAsk); !ok {
		t.Fatalf("the form did not ask: %T", hs.m.modal)
	}
	// Esc cancels: the form comes back, ready to start again.
	hs.key("esc")
	d, ok := hs.m.modal.(*newSession)
	if !ok || d.starting {
		t.Fatalf("after esc: %T", hs.m.modal)
	}
	for _, c := range hs.be.callList() {
		if strings.HasPrefix(c, "remember") {
			t.Fatalf("esc kept an answer: %s", c)
		}
	}
	hs.key("enter")
	hs.key("n")
	hasCall(t, hs.be, "remember atlas-dev /home/dev/src/app run=false")
	hasCall(t, hs.be, "new atlas-dev setup=skip")
	if hs.m.modal != nil || len(hs.be.attached) != 1 {
		t.Fatalf("after no: modal %T, attached %d", hs.m.modal, len(hs.be.attached))
	}
}

func TestQuickStartCancelSetupQuestion(t *testing.T) {
	hs := trustLab(t)
	hs.key("n", "enter", "esc")
	if hs.m.modal != nil || hs.m.prompt == nil || hs.m.prompt.starting {
		t.Fatalf("after esc: modal %T, prompt %+v", hs.m.modal, hs.m.prompt)
	}
	if len(hs.be.attached) != 0 {
		t.Fatal("a session started")
	}
}
