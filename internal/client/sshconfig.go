package client

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// SSHConfigHosts lists the Host aliases in the user's ~/.ssh/config, and in
// the files it includes, without wildcards, in the order they appear.
func SSHConfigHosts() []string {
	h := home()
	if h == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	readSSHConfig(filepath.Join(h, ".ssh", "config"), h, 0, seen, &out)
	return out
}

// readSSHConfig collects Host aliases without wildcards, following Include.
func readSSHConfig(path, home string, depth int, seen map[string]bool, out *[]string) {
	if depth > 4 {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// "Host a b", "Host=a" and "Host = a" are all valid.
		fields := strings.Fields(strings.Replace(line, "=", " ", 1))
		key := fields[0]
		fields = fields[1:]
		switch strings.ToLower(key) {
		case "host":
			for _, h := range fields {
				if !strings.ContainsAny(h, "*?!") && !seen[h] {
					seen[h] = true
					*out = append(*out, h)
				}
			}
		case "include":
			for _, p := range fields {
				if rest, ok := strings.CutPrefix(p, "~/"); ok {
					p = filepath.Join(home, rest)
				} else if !filepath.IsAbs(p) {
					p = filepath.Join(home, ".ssh", p)
				}
				matches, _ := filepath.Glob(p)
				for _, m := range matches {
					readSSHConfig(m, home, depth+1, seen, out)
				}
			}
		}
	}
}
