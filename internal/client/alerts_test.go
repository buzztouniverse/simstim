package client

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
)

func TestAlertsNeedANewHelper(t *testing.T) {
	c := &Conn{Host: Host{Name: "box"}, Features: []string{"ports", "folders"}}
	_, err := c.Alerts(context.Background())
	var ue *UserError
	if !errors.As(err, &ue) || !strings.Contains(ue.Msg, "too old for alerts") || !strings.Contains(ue.Hint, "host add box") {
		t.Errorf("an old helper gave %v", err)
	}
	c.Features = append(c.Features, "alerts")
	if c.needAlerts() != nil || !c.HasFeature("alerts") {
		t.Error("a helper with alerts was refused")
	}
}

func TestSecretFromCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	ctx := context.Background()
	if tok, err := SecretFromCommand(ctx, "printf 'tk_test\\n'"); err != nil || tok != "tk_test" {
		t.Errorf("SecretFromCommand = %q, %v", tok, err)
	}
	for _, line := range []string{"echo tk_test_leak; exit 1", "true", "printf 'tk_a\\ntk_b\\n'"} {
		_, err := SecretFromCommand(ctx, line)
		var ue *UserError
		if !errors.As(err, &ue) || ue.Hint == "" {
			t.Errorf("%q: err = %v, want a hint", line, err)
		}
		if err != nil && strings.Contains(err.Error(), "tk_") {
			t.Errorf("%q: the error shows the output: %v", line, err)
		}
	}
}
