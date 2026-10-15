package onboard

import (
	"strings"
	"testing"
)

func all() []string { return []string{Lima, Docker, E2B} }

func TestPlaces(t *testing.T) {
	on := map[string]bool{"ssh": true, "docker": true}
	ps := Places(Probe{GOOS: "darwin", Has: func(n string) bool { return on[n] }, SSHHosts: []string{"atlas", "gcp"}, Providers: all()})
	var order []string
	for _, p := range ps {
		order = append(order, p.Kind)
	}
	if strings.Join(order, " ") != "ssh lima docker e2b rent" {
		t.Fatalf("order = %v", order)
	}
	byKind := map[string]Place{}
	for _, p := range ps {
		byKind[p.Kind] = p
	}
	if p := byKind[SSH]; !p.Ready || p.Note != "2 found: atlas, gcp" || len(p.Hosts) != 2 {
		t.Errorf("ssh = %+v", p)
	}
	lima := byKind[Lima]
	if lima.Ready || lima.Note != "limactl is not on your PATH" || !hasRun(lima.Steps, "brew install lima") || hasRun(lima.Steps, "nix-env -i lima") {
		t.Errorf("lima on a Mac = %+v", lima)
	}
	if p := byKind[Docker]; !p.Ready || len(p.Steps) != 0 {
		t.Errorf("docker = %+v", p)
	}
	e2b := byKind[E2B]
	if e2b.Ready || !hasRun(e2b.Steps, "export E2B_API_KEY=e2b_your_key") || !strings.Contains(e2b.Steps[0].Say, "console.e2b.dev") {
		t.Errorf("e2b = %+v", e2b)
	}
	if p := byKind[Rent]; !p.Ready {
		t.Errorf("rent = %+v", p)
	}

	linux := Places(Probe{GOOS: "linux", Has: func(string) bool { return false }, E2BKey: "E2B_API_KEY", Providers: all()})
	l, _ := FindPlace(linux, Lima)
	if !hasRun(l.Steps, "nix-env -i lima") || hasRun(l.Steps, "sudo port install lima") {
		t.Errorf("lima on Linux = %+v", l)
	}
	if s, _ := FindPlace(linux, SSH); s.Ready || !hasRun(s.Steps, "sudo apt install openssh-client") {
		t.Errorf("ssh missing on Linux = %+v", s)
	}
	if e, _ := FindPlace(linux, E2B); !e.Ready || !strings.Contains(e.Note, "E2B_API_KEY") {
		t.Errorf("e2b with a key = %+v", e)
	}

	// A build without a provider does not offer it.
	none := Places(Probe{Has: func(string) bool { return true }})
	if len(none) != 2 {
		t.Fatalf("with no providers: %+v", none)
	}
	if _, ok := FindPlace(none, Lima); ok {
		t.Fatal("lima offered without its provider")
	}
}

func hasRun(ss []Step, cmd string) bool {
	for _, s := range ss {
		if s.Run == cmd {
			return true
		}
	}
	return false
}

func TestRentPlan(t *testing.T) {
	r := RentPlan{Addr: "203.0.113.10", User: "agents", Name: "my-server"}
	if !strings.Contains(r.KeySteps()[0].Run, "ssh-keygen -t ed25519") {
		t.Errorf("no key: %+v", r.KeySteps())
	}
	r.Key = "~/.ssh/id_ed25519.pub"
	if ks := r.KeySteps(); len(ks) != 1 || !strings.Contains(ks[0].Say, r.Key) {
		t.Errorf("a key: %+v", ks)
	}
	if !hasRun(r.ServerSteps(), "cat ~/.ssh/id_ed25519.pub") {
		t.Errorf("server steps: %+v", r.ServerSteps())
	}
	us := r.UserSteps()
	for _, want := range []string{"ssh root@203.0.113.10", `adduser --disabled-password --gecos "" agents`, `echo "agents ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/agents`} {
		if !hasRun(us, want) {
			t.Errorf("user steps lack %q", want)
		}
	}
	if r.Login() != "agents@203.0.113.10" {
		t.Errorf("login = %q", r.Login())
	}
	if got := r.ConfigLines(); got != "Host my-server\n    HostName 203.0.113.10\n    User agents\n" {
		t.Errorf("config lines = %q", got)
	}
	if (RentPlan{Addr: "198.51.100.4"}).Login() != "dev@198.51.100.4" {
		t.Error("the default user is not dev")
	}
	for s, ok := range map[string]bool{"203.0.113.10": true, "2001:db8::1": true, "vps.example.com": true, "": false, "-oProxyCommand=x": false, "a b": false, "a..b": false} {
		if ValidAddr(s) != ok {
			t.Errorf("ValidAddr(%q) = %v", s, !ok)
		}
	}
	for s, ok := range map[string]bool{"dev": true, "_svc": true, "Dev": false, "-x": false, "a b": false} {
		if ValidUser(s) != ok {
			t.Errorf("ValidUser(%q) = %v", s, !ok)
		}
	}
	for in, want := range map[string]string{"dev@203.0.113.10": "203.0.113.10", "box:2222": "box", "me@vps.example.com:22": "vps.example.com", "atlas": "atlas"} {
		if got := NameFor(in); got != want {
			t.Errorf("NameFor(%q) = %q", in, got)
		}
	}
}
