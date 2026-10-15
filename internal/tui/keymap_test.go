package tui

import "testing"

func TestWithKeymap(t *testing.T) {
	km := map[string]any{
		"session: delete": []any{"x", "Delete"},
		"session: rename": "",
		"host: add":       false,
	}
	cmds := withKeymap(registry(), km)
	find := func(name string) *command {
		for _, c := range cmds {
			if c.name() == name {
				return c
			}
		}
		t.Fatalf("no command %q", name)
		return nil
	}
	if d := find("session: delete"); len(d.keys) != 2 || d.keys[0].key != "x" || d.keys[1].key != "delete" {
		t.Errorf("delete keys = %+v", d.keys)
	}
	if r := find("session: rename"); len(r.keys) != 0 {
		t.Errorf("rename should be unbound, keys = %+v", r.keys)
	}
	if a := find("host: add"); len(a.keys) != 0 {
		t.Errorf("host add should be unbound, keys = %+v", a.keys)
	}
	if n := find("session: new"); len(n.keys) == 0 {
		t.Error("an untouched command lost its keys")
	}
}
