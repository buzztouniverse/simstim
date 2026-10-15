package agents

import "testing"

func TestRulesLoad(t *testing.T) {
	all, err := LoadRules()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"claude", "codex", "opencode", "gemini", "amp"} {
		if len(all[kind]) == 0 {
			t.Errorf("no rules for %s", kind)
		}
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		kind  string
		s     Screen
		state string
	}{
		{"claude", Screen{Lines: []string{"", "Bash command", " rm -rf build", "Do you want to proceed?", "❯ 1. Yes", "  2. No"}}, "needs_input"},
		{"claude", Screen{Lines: []string{"> fix the tests", "", "✻ Thinking… (12s · esc to interrupt)"}}, "working"},
		{"claude", Screen{Title: "⠂ Claude Code", Lines: []string{"> hi"}}, "working"},
		{"claude", Screen{Lines: []string{"> ", "? for shortcuts"}}, ""},
		{"codex", Screen{Title: "Action Required", Lines: nil}, "needs_input"},
		{"codex", Screen{Lines: []string{"Would you like to run the following command?", "$ make test"}}, "needs_input"},
		{"opencode", Screen{Lines: []string{"△ Permission required", "bash: ls"}}, "needs_input"},
		{"opencode", Screen{Lines: []string{"■■■■⬝⬝⬝⬝ working"}}, "working"},
		{"shell", Screen{Lines: []string{"$ ls"}}, ""},
	}
	for _, c := range cases {
		state, rule, ok := Detect(c.kind, c.s)
		if (c.state == "" && ok) || (c.state != "" && state != c.state) {
			t.Errorf("%s %v: got %q (rule %q, ok %v), want %q", c.kind, c.s.Lines, state, rule, ok, c.state)
		}
	}
}
