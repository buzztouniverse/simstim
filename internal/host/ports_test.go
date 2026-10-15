package host

import "testing"

func TestParseHexAddr(t *testing.T) {
	cases := map[string]struct {
		addr string
		port int
	}{
		"0100007F:0BB8":                         {"127.0.0.1", 3000},
		"00000000:1F90":                         {"0.0.0.0", 8080},
		"00000000000000000000000000000000:1538": {"::", 5432},
		"00000000000000000000000001000000:0050": {"::1", 80},
	}
	for in, want := range cases {
		addr, port := parseHexAddr(in)
		if addr != want.addr || port != want.port {
			t.Errorf("parseHexAddr(%q) = %s:%d, want %s:%d", in, addr, port, want.addr, want.port)
		}
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"Fix auth bug!": "fix-auth-bug", "": "task", "  --x--  ": "x"} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
