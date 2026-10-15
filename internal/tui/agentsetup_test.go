package tui

import (
	"strings"
	"testing"

	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/client"
)

func TestNewSessionGetsTheHostsSecrets(t *testing.T) {
	hs := home(t, 80, 24)
	hs.be.secrets = map[string]api.Secret{"ANTHROPIC_API_KEY": "sk-test-value"}
	hs.key("n", "enter")
	hasCall(t, hs.be, "secrets atlas-dev")
	// The fake records the value as fmt prints it: redacted.
	hasCall(t, hs.be, "new atlas-dev secret ANTHROPIC_API_KEY=[redacted]")
	if len(hs.be.attached) != 1 {
		t.Fatal("the session did not start")
	}
	for _, c := range hs.be.callList() {
		if strings.Contains(c, "sk-test-value") {
			t.Fatalf("a secret was printed: %s", c)
		}
	}
}

func TestNewSessionSecretError(t *testing.T) {
	hs := home(t, 80, 24)
	hs.be.secretErr = &client.UserError{Msg: "the command of the secret ANTHROPIC_API_KEY failed (exit code 1)", Hint: "Run the command of ANTHROPIC_API_KEY from config.toml yourself, to see what it says."}
	hs.key("n", "enter")
	for _, c := range hs.be.callList() {
		if strings.HasPrefix(c, "new atlas-dev") {
			t.Fatalf("a session started without its secret: %s", c)
		}
	}
	p := hs.m.prompt
	if p == nil || p.err == nil || p.starting {
		t.Fatalf("the error is not on the prompt: %+v", p)
	}
	if out := plainScreen(hs.m); !strings.Contains(out, "secret ANTHROPIC_API_KEY failed") {
		t.Fatalf("the error is not shown:\n%s", out)
	}
}

func TestMissingSecretsShow(t *testing.T) {
	be := labBackend()
	st := be.state["atlas-dev"]
	st.sessions[2].MissingSecrets = []string{"OPENAI_API_KEY"}
	hs := newHarness(t, be, 100, 24)
	hs.m.motion = false
	hs.key("enter", "down")
	if hs.m.sel.id != "s3" {
		t.Fatalf("selected %v", hs.m.sel)
	}
	if out := plainScreen(hs.m); !strings.Contains(out, "started without its secrets: OPENAI_API_KEY") {
		t.Fatalf("the launcher does not say the session has no secrets:\n%s", out)
	}
}
