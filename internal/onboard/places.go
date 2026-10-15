// Package onboard is the guided setup's engine. It knows where agents can
// run on this computer and what each place needs, the walk-through for a
// server the user rents, how to make a host with a provider and clean up
// after a stop, and the agent part of a setup: the checklist, the installs
// and the logins, and what was done.
//
// The setup command (internal/cli) and the launcher (internal/tui) both drive
// it. They hold the screens and the prompts; the decisions live here. The
// design is in docs/contributing/decisions/0020-guided-setup.md.
package onboard

import (
	"fmt"
	"strings"

	"github.com/iamlakshya/simstim/internal/provider/e2b"
)

// The places agents can run, as Place.Kind names them. Lima, Docker and E2B
// are also the names of their providers.
const (
	SSH    = "ssh"
	Lima   = "lima"
	Docker = "docker"
	E2B    = "e2b"
	Rent   = "rent"
)

// Place is one answer to "where will agents run?".
type Place struct {
	Kind  string
	Title string
	// Ready is true when this computer can use the place now.
	Ready bool
	// Note says what the place gives when it is ready, or what is missing.
	Note string
	// Steps make the place ready, in order. They are empty when it is.
	Steps []Step
	// Hosts are the Host aliases in the user's SSH config, for SSH.
	Hosts []string
}

// Step is one thing for the user to do: words, and maybe a command to run.
type Step struct {
	Say string
	Run string
}

// Probe is what Places reads from this computer. System fills it in from
// the real one; tests fill it in by hand.
type Probe struct {
	// GOOS picks the install steps: "darwin", "linux", or anything else
	// for both.
	GOOS string
	// Has reports whether a program is on PATH.
	Has func(name string) bool
	// SSHHosts are the Host aliases in ~/.ssh/config.
	SSHHosts []string
	// E2BKey says where an E2B API key comes from, or "" (never the key).
	E2BKey string
	// Providers names the providers this build can make hosts with.
	Providers []string
}

// Places lists where agents can run, in the order the guided setup offers
// them, and says for each whether this computer can use it now, and if not,
// how to make it possible.
func Places(p Probe) []Place {
	has := func(name string) bool { return p.Has != nil && p.Has(name) }
	built := func(kind string) bool {
		for _, k := range p.Providers {
			if k == kind {
				return true
			}
		}
		return false
	}
	var out []Place

	ssh := Place{Kind: SSH, Title: "A host in your SSH config", Ready: has("ssh"), Hosts: p.SSHHosts}
	switch {
	case !ssh.Ready:
		ssh.Note = "ssh is not on your PATH"
		ssh.Steps = sshSteps(p.GOOS)
	case len(p.SSHHosts) == 0:
		ssh.Note = "none found; you can type user@host"
	default:
		ssh.Note = fmt.Sprintf("%d found: %s", len(p.SSHHosts), strings.Join(p.SSHHosts, ", "))
	}
	out = append(out, ssh)

	if built(Lima) {
		pl := Place{Kind: Lima, Title: "A local VM, with Lima", Ready: has("limactl"),
			Note: "free, on this computer"}
		if !pl.Ready {
			pl.Note, pl.Steps = "limactl is not on your PATH", limaSteps(p.GOOS)
		}
		out = append(out, pl)
	}
	if built(Docker) {
		pl := Place{Kind: Docker, Title: "A container, for testing (Docker)", Ready: has("docker"),
			Note: "no systemd, so no memory caps"}
		if !pl.Ready {
			pl.Note, pl.Steps = "docker is not on your PATH", dockerSteps(p.GOOS)
		}
		out = append(out, pl)
	}
	if built(E2B) {
		pl := Place{Kind: E2B, Title: "A cloud sandbox, with E2B", Ready: p.E2BKey != "",
			Note: "key from " + p.E2BKey}
		if !pl.Ready {
			pl.Note, pl.Steps = "no E2B API key found", e2bSteps()
		}
		out = append(out, pl)
	}
	out = append(out, Place{Kind: Rent, Title: "A server I will rent", Ready: true,
		Note: "step by step, with checks"})
	return out
}

// FindPlace returns the place of the given kind.
func FindPlace(places []Place, kind string) (Place, bool) {
	for _, p := range places {
		if p.Kind == kind {
			return p, true
		}
	}
	return Place{}, false
}

func onMac(goos string) bool   { return goos == "darwin" }
func onLinux(goos string) bool { return goos == "linux" }
func either(goos string) bool  { return !onMac(goos) && !onLinux(goos) }

func sshSteps(goos string) []Step {
	var s []Step
	if onLinux(goos) || either(goos) {
		s = append(s, Step{Say: "Install the OpenSSH client from your distribution's packages, for example:", Run: "sudo apt install openssh-client"})
	}
	if onMac(goos) || either(goos) {
		s = append(s, Step{Say: "macOS has ssh in /usr/bin. Check that your PATH includes it."})
	}
	return s
}

// limaSteps come from Lima's install guide, https://lima-vm.io/docs/installation/.
func limaSteps(goos string) []Step {
	var s []Step
	if onMac(goos) || either(goos) {
		s = append(s,
			Step{Say: "On macOS, with Homebrew:", Run: "brew install lima"},
			Step{Say: "or with MacPorts:", Run: "sudo port install lima"})
	}
	if onLinux(goos) || either(goos) {
		s = append(s,
			Step{Say: "On Linux, with Homebrew:", Run: "brew install lima"},
			Step{Say: "or with Nix:", Run: "nix-env -i lima"},
			Step{Say: "On Linux, Lima runs VMs with QEMU: install it from your distribution's packages."})
	}
	return append(s, Step{Say: "Other ways, such as the release archive, are in Lima's guide: https://lima-vm.io/docs/installation/"})
}

// dockerSteps come from Docker's install guides.
func dockerSteps(goos string) []Step {
	var s []Step
	if onMac(goos) || either(goos) {
		s = append(s, Step{Say: "On macOS, install Docker Desktop and start it: https://docs.docker.com/desktop/setup/install/mac-install/"})
	}
	if onLinux(goos) || either(goos) {
		s = append(s,
			Step{Say: "On Linux, install Docker Engine for your distribution: https://docs.docker.com/engine/install/"},
			Step{Say: "Then let your user run it without sudo: https://docs.docker.com/engine/install/linux-postinstall/"})
	}
	return append(s, Step{Say: "Check that it answers:", Run: "docker info"})
}

// e2bSteps come from E2B's docs, https://e2b.dev/docs/api-key. Signing up
// is the user's own step: the setup never makes an account.
func e2bSteps() []Step {
	return []Step{
		{Say: "Make an E2B account, then copy an API key from its console: https://console.e2b.dev/?tab=keys"},
		{Say: "Put the key in your shell's environment, for example in ~/.zshrc:", Run: "export E2B_API_KEY=e2b_your_key"},
		{Say: "or log in with the E2B CLI:", Run: "e2b auth login"},
		{Say: "Agents want 2 CPUs and 4 GiB, which belong to an E2B template. Build it once: write `FROM e2bdev/base` into e2b.Dockerfile, then run:",
			Run: fmt.Sprintf("e2b template create %s --cpu-count 2 --memory-mb 4096", e2b.DefaultTemplate)},
	}
}
