package onboard

import (
	"fmt"
	"regexp"
	"strings"
)

// The walk-through for a server the user rents, from
// docs/tutorials/first-cloud-host.md, one step at a time. It never rents
// anything, and it changes ~/.ssh/config only after it shows the exact
// lines and the user says yes.

// Defaults for a rented server: the user agents run as, and the host's name.
const (
	RentUser = "dev"
	RentName = "my-server"
)

// The rent steps, in order.
const (
	RentKey = iota
	RentServer
	RentUserStep
	RentNameStep
)

// RentTitles are the steps' titles, by step.
var RentTitles = []string{"An SSH key", "A server", "A user for your agents", "A name for the server"}

// RentPlan is what the user has told the walk-through so far.
type RentPlan struct {
	// Key is the public key file found in ~/.ssh, or "".
	Key string
	// Addr is the server's address.
	Addr string
	// User is the user that agents run as, on the server.
	User string
	// Name is the host's name, in ~/.ssh/config and in hosts.toml.
	Name string
}

// KeyCommand makes an SSH key, as the tutorial does.
const KeyCommand = `ssh-keygen -t ed25519 -C "my laptop"`

// KeyArgs are KeyCommand's words, to run it without a shell.
var KeyArgs = []string{"ssh-keygen", "-t", "ed25519", "-C", "my laptop"}

// KeySteps say what the key step finds, or how to make a key.
func (r RentPlan) KeySteps() []Step {
	if r.Key != "" {
		return []Step{{Say: "Found your public key, " + r.Key + ". The server gets this one."}}
	}
	return []Step{
		{Say: "There is no SSH key in ~/.ssh yet. Make one on this computer:", Run: KeyCommand},
		{Say: "Press Enter to keep the default file, and give the key a passphrase. A Mac keeps the passphrase in its keychain."},
	}
}

// ServerSteps say how to rent the server. The user does it in the
// provider's own console.
func (r RentPlan) ServerSteps() []Step {
	key := r.Key
	if key == "" {
		key = "~/.ssh/id_ed25519.pub"
	}
	return []Step{
		{Say: "In your provider's console (Hetzner Cloud, DigitalOcean, Vultr, Linode, AWS Lightsail or another), create a server:"},
		{Say: "Image: Ubuntu 24.04 LTS or Debian 13. Size: at least 2 vCPUs and 4 GB of memory."},
		{Say: "SSH key: paste your public key. Print it with:", Run: "cat " + key},
		{Say: "The server costs money while it exists. Nothing here rents or deletes it for you."},
	}
}

// UserSteps make a user for agents on the new server, as root.
func (r RentPlan) UserSteps() []Step {
	u := r.user()
	return []Step{
		{Say: "Agents should not run as root. Log in once as root, and make a user with your key:"},
		{Run: "ssh root@" + r.Addr},
		{Run: `adduser --disabled-password --gecos "" ` + u},
		{Run: "usermod -aG sudo " + u},
		{Run: fmt.Sprintf(`echo "%s ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/%s`, u, u)},
		{Run: fmt.Sprintf("install -d -m 700 -o %s -g %s /home/%s/.ssh", u, u, u)},
		{Run: fmt.Sprintf("install -m 600 -o %s -g %s ~/.ssh/authorized_keys /home/%s/.ssh/", u, u, u)},
		{Run: "exit"},
		{Say: "Passwordless sudo lets agents install packages. Leave out the echo line to type a password for sudo yourself."},
	}
}

func (r RentPlan) user() string {
	if r.User == "" {
		return RentUser
	}
	return r.User
}

// Login is the SSH target of the user step's check: user@address.
func (r RentPlan) Login() string { return r.user() + "@" + r.Addr }

// ConfigLines are the exact lines that give the server its name in
// ~/.ssh/config.
func (r RentPlan) ConfigLines() string {
	return fmt.Sprintf("Host %s\n    HostName %s\n    User %s\n", r.Name, r.Addr, r.user())
}

var (
	addrRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.:-]*$`)
	userRE = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
)

// ValidAddr reports whether s can be a server's address: an IPv4 or IPv6
// address, or a DNS name.
func ValidAddr(s string) bool { return addrRE.MatchString(s) && !strings.Contains(s, "..") }

// ValidUser reports whether s can be a Linux user name.
func ValidUser(s string) bool { return userRE.MatchString(s) }

// NameFor is a host's name from its SSH target: user@host:port gives host.
func NameFor(target string) string {
	t := target
	if i := strings.LastIndex(t, "@"); i >= 0 {
		t = t[i+1:]
	}
	if i := strings.Index(t, ":"); i >= 0 {
		t = t[:i]
	}
	return t
}
