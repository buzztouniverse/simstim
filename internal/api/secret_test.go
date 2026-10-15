package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestSecretNeverPrints(t *testing.T) {
	const value = "sk-test-0123456789"
	a := NewArgs{Agent: "claude", Secrets: map[string]Secret{"OPENAI_API_KEY": value}}
	var logged strings.Builder
	log.New(&logged, "", 0).Printf("new %v %+v", a, a)
	for _, out := range []string{
		fmt.Sprint(a), fmt.Sprintf("%v %+v %#v %s %q %x %X %d", a, a, a, a.Secrets["OPENAI_API_KEY"], a.Secrets["OPENAI_API_KEY"],
			a.Secrets["OPENAI_API_KEY"], a.Secrets["OPENAI_API_KEY"], a.Secrets["OPENAI_API_KEY"]),
		fmt.Sprint(a.Secrets), logged.String(),
	} {
		if strings.Contains(out, value) || strings.Contains(out, fmt.Sprintf("%x", value)) {
			t.Fatalf("a secret was printed: %s", out)
		}
		if !strings.Contains(out, Redacted) {
			t.Fatalf("no redaction mark in %s", out)
		}
	}
	// JSON carries the value: it is how the value reaches the host.
	b, err := json.Marshal(a)
	if err != nil || !strings.Contains(string(b), `"secrets":{"OPENAI_API_KEY":"`+value+`"}`) {
		t.Fatalf("json = %s, %v", b, err)
	}
	var back NewArgs
	if err := json.Unmarshal(b, &back); err != nil || string(back.Secrets["OPENAI_API_KEY"]) != value {
		t.Fatalf("round trip = %v, %v", back.Secrets, err)
	}
}
