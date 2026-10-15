package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/iamlakshya/simstim/internal/alerts"
	"github.com/iamlakshya/simstim/internal/api"
	"github.com/iamlakshya/simstim/internal/paths"
)

// alertsRequest answers the bridge's alerts.* requests.
func alertsRequest(op string, args json.RawMessage) (any, error) {
	switch op {
	case "alerts.get":
		return AlertsGet(), nil
	case "alerts.set":
		var a api.AlertsSetArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, UserError("bad_request", "the alerts config did not arrive whole", "Try again.")
		}
		return AlertsSet(a)
	case "alerts.test":
		var a api.AlertsTestArgs
		_ = json.Unmarshal(args, &a)
		return AlertsTest(a)
	case "alerts.log":
		var a api.AlertsLogArgs
		_ = json.Unmarshal(args, &a)
		return AlertsLog(a)
	case "alerts.off":
		return AlertsOff()
	}
	return nil, UserError("unknown_op", "unknown request "+op, "Update the host helper.")
}

// AlertsGet describes this host's alerts file, with its secrets hidden.
func AlertsGet() api.AlertsInfo {
	path := paths.AlertsConfig()
	info := api.AlertsInfo{Path: homeRelative(path)}
	c, exists, err := alerts.Read(path)
	info.Exists = exists
	if err == nil && exists {
		r := alerts.Redact(alerts.WithDefaults(c))
		info.Config = &r
		_, err = alerts.Load(path)
	}
	if err != nil {
		info.Error, info.Hint = errorParts(err)
	}
	return info
}

// AlertsSet writes this host's alerts file: the whole of it, or with Add,
// new destinations and settings over the file's.
func AlertsSet(a api.AlertsSetArgs) (api.AlertsInfo, error) {
	path := paths.AlertsConfig()
	c := a.Config
	if a.Add {
		old, exists, err := alerts.Read(path)
		if err != nil {
			msg, _ := errorParts(err)
			return api.AlertsInfo{}, UserError("alerts", "cannot add to the alerts file: "+msg, "Write it whole, without --add, or fix "+homeRelative(path)+".")
		}
		if exists {
			c = alerts.Merge(old, a.Config)
		}
	}
	if err := alerts.Save(path, c); err != nil {
		return api.AlertsInfo{}, alertsError(err)
	}
	return AlertsGet(), nil
}

// AlertsTest sends a test event to every destination at once, and reports
// what each did. It ignores the delay and the limits.
func AlertsTest(a api.AlertsTestArgs) ([]api.AlertResult, error) {
	home, _ := os.UserHomeDir()
	ev, err := alerts.TestEvent(a.Event, home)
	if err != nil {
		return nil, alertsError(err)
	}
	// Commands get the login environment, as they do in a session.
	env, err := LoginEnv(5 * time.Second)
	if err != nil {
		env = os.Environ()
	}
	res, err := alerts.SendNow(alerts.Options{Env: env, Dir: home}, ev)
	if err != nil {
		return nil, alertsError(err)
	}
	return res, nil
}

// AlertsLog returns the newest lines of the delivery log, oldest first.
func AlertsLog(a api.AlertsLogArgs) ([]api.AlertLogEntry, error) {
	n := a.Lines
	if n <= 0 {
		n = 20
	}
	l := &alerts.DeliveryLog{Path: paths.AlertsLog(), Lock: paths.AlertsRate()}
	out, err := l.Tail(min(n, 1000))
	if err != nil {
		return nil, fmt.Errorf("read the alerts log: %w", err)
	}
	if out == nil {
		out = []api.AlertLogEntry{}
	}
	return out, nil
}

// AlertsOff removes this host's alerts file, which turns alerts off.
func AlertsOff() (map[string]bool, error) {
	err := os.Remove(paths.AlertsConfig())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return map[string]bool{"removed": false}, nil
	case err != nil:
		return nil, fmt.Errorf("remove the alerts file: %w", err)
	}
	return map[string]bool{"removed": true}, nil
}

// alertsError turns an alerts error into one the client shows with its hint.
func alertsError(err error) error {
	msg, hint := errorParts(err)
	return UserError("alerts", msg, hint)
}

func errorParts(err error) (msg, hint string) {
	var e *alerts.Error
	if errors.As(err, &e) {
		return e.Msg, e.Hint
	}
	return err.Error(), ""
}

// homeRelative writes a path under the home folder as ~/….
func homeRelative(p string) string {
	if h, err := os.UserHomeDir(); err == nil && h != "/" {
		if rest, ok := strings.CutPrefix(p, h+"/"); ok {
			return "~/" + rest
		}
	}
	return p
}
