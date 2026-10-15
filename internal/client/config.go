// Package client is the laptop side: hosts, SSH, installing the host helper,
// the bridge connection and the attach pump.
package client

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// Host is one saved host.
type Host struct {
	// Name is how the user refers to the host.
	Name string `toml:"name"`
	// SSH is the ssh destination: a Host alias from an SSH config, or user@host[:port].
	SSH string `toml:"ssh"`
	// SSHConfig is an extra SSH config file to include, for example a repo's own config.
	SSHConfig string `toml:"ssh_config,omitempty"`
	// Labels are free-form, for example provider = "gcp".
	Labels map[string]string `toml:"labels,omitempty"`
	// Provider is an optional script that starts and stops the host.
	Provider *Provider `toml:"provider,omitempty"`
}

// Settings are the user's preferences.
type Settings struct {
	UI struct {
		Motion string `toml:"motion"`
	} `toml:"ui"`
	Attach struct {
		DetachKey     string `toml:"detach_key"`
		ImagePasteKey string `toml:"image_paste_key"`
		Scrollback    string `toml:"scrollback"`
	} `toml:"attach"`
	// Keymap rebinds home-screen commands by name ("session: delete"). A value is
	// a key, a list of keys, or "" to unbind.
	Keymap map[string]any `toml:"keymap"`
}

type hostsFile struct {
	Hosts []Host `toml:"host"`
}

// ConfigDir is $XDG_CONFIG_HOME/<name>, or ~/.config/<name> (on macOS too).
func ConfigDir() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, brand.Name)
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config", brand.Name)
}

// CacheDir is $XDG_CACHE_HOME/<name>, or ~/.cache/<name>.
func CacheDir() string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, brand.Name)
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".cache", brand.Name)
}

// StateDir is $XDG_STATE_HOME/<name>, or ~/.local/state/<name>.
func StateDir() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, brand.Name)
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".local", "state", brand.Name)
}

func hostsPath() string    { return filepath.Join(ConfigDir(), "hosts.toml") }
func settingsPath() string { return filepath.Join(ConfigDir(), "config.toml") }

// LoadHosts reads the saved hosts.
func LoadHosts() ([]Host, error) {
	var f hostsFile
	if _, err := toml.DecodeFile(hostsPath(), &f); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", hostsPath(), err)
	}
	return f.Hosts, nil
}

// SaveHosts writes the saved hosts.
func SaveHosts(hosts []Host) error {
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return err
	}
	tmp := hostsPath() + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "# Hosts for %s. Edit with `%s host`.\n\n", brand.DisplayName, brand.Name)
	if err := toml.NewEncoder(f).Encode(hostsFile{Hosts: hosts}); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, hostsPath())
}

// FindHost returns the saved host with the given name.
func FindHost(name string) (Host, error) {
	hosts, err := LoadHosts()
	if err != nil {
		return Host{}, err
	}
	for _, h := range hosts {
		if h.Name == name {
			return h, nil
		}
	}
	return Host{}, &UserError{Msg: fmt.Sprintf("no host named %q", name), Hint: fmt.Sprintf("%s host add %s", brand.Name, name)}
}

var hostNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$`)

// ValidHostName reports whether n can name a host.
func ValidHostName(n string) bool { return hostNameRE.MatchString(n) }

// LoadSettings reads the settings, with defaults for anything missing.
func LoadSettings() Settings {
	var s Settings
	_, _ = toml.DecodeFile(settingsPath(), &s)
	if s.UI.Motion == "" {
		s.UI.Motion = "auto"
	}
	if s.Attach.DetachKey == "" {
		s.Attach.DetachKey = "ctrl+\\"
	}
	if s.Attach.ImagePasteKey == "" {
		// Windows Terminal keeps Ctrl+V for itself.
		s.Attach.ImagePasteKey = "ctrl+v"
		if runtime.GOOS == "windows" {
			s.Attach.ImagePasteKey = "alt+v"
		}
	}
	if s.Attach.Scrollback == "" {
		s.Attach.Scrollback = "keep"
	}
	return s
}

// UserError is an error with a suggested next step.
type UserError struct {
	Msg  string
	Hint string
}

func (e *UserError) Error() string { return e.Msg }

// SplitTarget splits "host/session" into its parts.
func SplitTarget(t string) (host, session string, err error) {
	host, session, ok := strings.Cut(t, "/")
	if !ok || host == "" || session == "" {
		return "", "", &UserError{Msg: fmt.Sprintf("%q is not host/session", t), Hint: brand.Name + " ls"}
	}
	return host, session, nil
}
