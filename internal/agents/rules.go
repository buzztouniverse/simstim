package agents

import (
	"embed"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

// Screen rules read an agent's visible screen to tell whether it is working or
// waiting for the user. They are data, one TOML file per agent, so they can be
// fixed without code when an agent changes its wording.
//
// A rule has a state ("working", "needs_input" or "idle"), a priority (higher
// first), a region and conditions. The first rule whose conditions hold wins.
//
// Regions: "title" (the window title), "screen" (every line), "bottom:N" and
// "top:N" (the last or first N non-empty lines).
//
// Conditions: contains (every string must appear, ignoring case), regex (any
// pattern matches the region), line_regex (any pattern matches any line),
// any (at least one sub-condition holds) and not (no sub-condition holds).

//go:embed rules/*.toml
var ruleFiles embed.FS

// Condition is a test on a region's text.
type Condition struct {
	Contains  []string    `toml:"contains"`
	Regex     []string    `toml:"regex"`
	LineRegex []string    `toml:"line_regex"`
	Any       []Condition `toml:"any"`
	Not       []Condition `toml:"not"`

	re     []*regexp.Regexp
	lineRe []*regexp.Regexp
}

// Rule is one screen rule.
type Rule struct {
	ID       string `toml:"id"`
	State    string `toml:"state"`
	Priority int    `toml:"priority"`
	Region   string `toml:"region"`
	Condition
}

type ruleFile struct {
	Agent string `toml:"agent"`
	Rules []Rule `toml:"rules"`
}

// Screen is what the rules look at.
type Screen struct {
	Title string
	Lines []string
}

var (
	rulesOnce sync.Once
	rules     map[string][]Rule
	errRules  error
)

// LoadRules parses the embedded rule files. It is called once.
func LoadRules() (map[string][]Rule, error) {
	rulesOnce.Do(func() {
		rules = map[string][]Rule{}
		entries, err := ruleFiles.ReadDir("rules")
		if err != nil {
			errRules = err
			return
		}
		for _, e := range entries {
			b, err := ruleFiles.ReadFile("rules/" + e.Name())
			if err != nil {
				errRules = err
				return
			}
			var f ruleFile
			if _, err := toml.Decode(string(b), &f); err != nil {
				errRules = fmt.Errorf("%s: %w", e.Name(), err)
				return
			}
			for i := range f.Rules {
				if err := f.Rules[i].compile(); err != nil {
					errRules = fmt.Errorf("%s rule %s: %w", e.Name(), f.Rules[i].ID, err)
					return
				}
			}
			sort.SliceStable(f.Rules, func(i, j int) bool { return f.Rules[i].Priority > f.Rules[j].Priority })
			rules[f.Agent] = f.Rules
		}
	})
	return rules, errRules
}

func (c *Condition) compile() error {
	for _, p := range c.Regex {
		re, err := regexp.Compile(p)
		if err != nil {
			return fmt.Errorf("regex %q: %w", p, err)
		}
		c.re = append(c.re, re)
	}
	for _, p := range c.LineRegex {
		re, err := regexp.Compile(p)
		if err != nil {
			return fmt.Errorf("line regex %q: %w", p, err)
		}
		c.lineRe = append(c.lineRe, re)
	}
	for i := range c.Any {
		if err := c.Any[i].compile(); err != nil {
			return err
		}
	}
	for i := range c.Not {
		if err := c.Not[i].compile(); err != nil {
			return err
		}
	}
	return nil
}

// Detect returns the state the agent's screen shows, and the rule that said so.
// ok is false when no rule matched, or the agent has no rules.
func Detect(kind string, s Screen) (state, rule string, ok bool) {
	all, err := LoadRules()
	if err != nil {
		return "", "", false
	}
	for _, r := range all[kind] {
		lines := region(r.Region, s)
		text := strings.Join(lines, "\n")
		if r.holds(text, lines) {
			return r.State, r.ID, true
		}
	}
	return "", "", false
}

func region(name string, s Screen) []string {
	if name == "title" {
		return []string{s.Title}
	}
	var nonEmpty []string
	for _, l := range s.Lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}
	kind, n, _ := strings.Cut(name, ":")
	count, _ := strconv.Atoi(n)
	switch kind {
	case "bottom":
		if count > 0 && count < len(nonEmpty) {
			return nonEmpty[len(nonEmpty)-count:]
		}
		return nonEmpty
	case "top":
		if count > 0 && count < len(nonEmpty) {
			return nonEmpty[:count]
		}
		return nonEmpty
	default:
		return s.Lines
	}
}

func (c *Condition) holds(text string, lines []string) bool {
	lower := strings.ToLower(text)
	for _, s := range c.Contains {
		if !strings.Contains(lower, strings.ToLower(s)) {
			return false
		}
	}
	if len(c.re) > 0 {
		hit := false
		for _, re := range c.re {
			if re.MatchString(text) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	if len(c.lineRe) > 0 {
		hit := false
		for _, re := range c.lineRe {
			for _, l := range lines {
				if re.MatchString(l) {
					hit = true
					break
				}
			}
			if hit {
				break
			}
		}
		if !hit {
			return false
		}
	}
	if len(c.Any) > 0 {
		hit := false
		for i := range c.Any {
			if c.Any[i].holds(text, lines) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	for i := range c.Not {
		if c.Not[i].holds(text, lines) {
			return false
		}
	}
	return true
}
