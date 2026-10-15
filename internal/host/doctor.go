package host

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/iAmLakshya/simstim/internal/agents"
	"github.com/iAmLakshya/simstim/internal/api"
	"github.com/iAmLakshya/simstim/internal/paths"
	"github.com/iAmLakshya/simstim/internal/version"
)

type Check = api.Check

type HostInfo = api.HostInfo

// Doctor inspects the host.
func Doctor() HostInfo {
	hn, _ := os.Hostname()
	u, _ := user.Current()
	info := HostInfo{Hostname: hn, OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU(),
		MemTotal: MemTotal(), Version: version.Version, Agents: map[string]string{}, MemoryMax: DefaultMemoryMax()}
	if u != nil {
		info.User = u.Username
	}
	if b, err := exec.Command("uname", "-r").Output(); err == nil {
		info.Kernel = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(l, "PRETTY_NAME="); ok {
				info.Distro = strings.Trim(v, `"`)
			}
		}
	}
	if h, err := os.UserHomeDir(); err == nil {
		info.DiskFree = diskFree(h)
	}
	login, err := LoginEnv(15 * time.Second)
	if err != nil {
		login = os.Environ()
	}
	path := envValue(login, "PATH")
	for _, a := range agents.Table {
		for _, b := range a.Binaries {
			if b == "$SHELL" {
				info.Agents[a.Kind] = LoginShell()
				break
			}
			if p, err := lookPath(b, path); err == nil {
				info.Agents[a.Kind] = p
				break
			}
		}
	}
	sd := SystemdUser()
	info.Checks = append(info.Checks, Check{Name: "systemd user manager", OK: sd,
		Detail: map[bool]string{true: "sessions run as user services with memory caps", false: "sessions run as detached processes without memory caps"}[sd]})
	if sd {
		linger := lingerOn()
		info.Checks = append(info.Checks, Check{Name: "linger", OK: linger,
			Detail: map[bool]string{true: "sessions survive logout", false: "sessions stop when your last login ends"}[linger],
			Fix:    "loginctl enable-linger $USER"})
		mem := memoryDelegated()
		info.Checks = append(info.Checks, Check{Name: "memory caps", OK: mem,
			Detail: map[bool]string{true: "cgroup v2 memory controller is delegated", false: "the memory controller is not delegated to the user manager"}[mem]})
	}
	gt := HasTerminfo("xterm-ghostty")
	info.Checks = append(info.Checks, Check{Name: "Ghostty terminfo", OK: gt,
		Detail: map[bool]string{true: "xterm-ghostty is installed", false: "xterm-ghostty is missing; sessions fall back to xterm-256color"}[gt],
		Fix:    "infocmp -x xterm-ghostty | ssh <host> tic -x -"})
	info.Checks = append(info.Checks, Check{Name: "runtime folder", OK: true, Detail: paths.RuntimeDir()})
	info.Checks = append(info.Checks, IdleDoctor()...)
	return info
}

func lingerOn() bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	out, err := exec.Command("loginctl", "show-user", u.Username, "-p", "Linger", "--value").Output()
	return err == nil && strings.TrimSpace(string(out)) == "yes"
}

func memoryDelegated() bool {
	uid := os.Getuid()
	b, err := os.ReadFile(fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/cgroup.controllers", uid, uid))
	return err == nil && strings.Contains(string(b), "memory")
}

type SetupArgs = api.SetupArgs

type SetupResult = api.SetupResult

// Setup turns on linger and installs terminfo entries.
func Setup(a SetupArgs) SetupResult {
	var res SetupResult
	if a.Linger && SystemdUser() {
		if lingerOn() {
			res.Steps = append(res.Steps, Check{Name: "linger", OK: true, Detail: "already on"})
		} else {
			u, _ := user.Current()
			err := exec.Command("loginctl", "enable-linger", u.Username).Run()
			if err != nil {
				err = exec.Command("sudo", "-n", "loginctl", "enable-linger", u.Username).Run()
			}
			ok := err == nil && lingerOn()
			res.Steps = append(res.Steps, Check{Name: "linger", OK: ok,
				Detail: map[bool]string{true: "turned on", false: "could not turn it on"}[ok],
				Fix:    "sudo loginctl enable-linger " + u.Username})
		}
	}
	if a.Linger {
		res.Steps = append(res.Steps, InstallRestoreUnit())
	}
	for name, src := range a.Terminfo {
		if HasTerminfo(name) {
			res.Steps = append(res.Steps, Check{Name: "terminfo " + name, OK: true, Detail: "already installed"})
			continue
		}
		f, err := os.CreateTemp("", "terminfo-*")
		if err == nil {
			_, _ = f.WriteString(src)
			f.Close()
			home, _ := os.UserHomeDir()
			out, terr := exec.Command("tic", "-x", "-o", filepath.Join(home, ".terminfo"), f.Name()).CombinedOutput()
			os.Remove(f.Name())
			err = terr
			if err != nil {
				err = fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
			}
		}
		ok := err == nil && HasTerminfo(name)
		d := "installed"
		if !ok {
			d = fmt.Sprintf("could not install: %v", err)
		}
		res.Steps = append(res.Steps, Check{Name: "terminfo " + name, OK: ok, Detail: d})
	}
	return res
}
