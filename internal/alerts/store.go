package alerts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/iamlakshya/simstim/internal/api"
)

// HostLimit is the most events one host sends in a minute, across all its sessions.
const HostLimit = 20

// lockWait is the longest a holder waits for the shared files.
const lockWait = 2 * time.Second

// Limiter caps how many events a whole host sends.
type Limiter interface {
	// Allow reports whether one more event may go out at now, and counts it if so.
	Allow(now time.Time) bool
}

// FileLimiter counts recent events in a file every holder on the host shares,
// under a lock. If the file cannot be used, it lets events through: an alert
// matters more than the noise limit.
type FileLimiter struct {
	Path   string
	Max    int
	Window time.Duration
}

// NewFileLimiter returns a limiter of HostLimit events a minute, counted in path.
func NewFileLimiter(path string) *FileLimiter {
	return &FileLimiter{Path: path, Max: HostLimit, Window: time.Minute}
}

// Allow implements Limiter.
func (l *FileLimiter) Allow(now time.Time) bool {
	ok := true
	ctx, cancel := context.WithTimeout(context.Background(), lockWait)
	defer cancel()
	err := withLock(ctx, l.Path, func(f *os.File) error {
		data, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", l.Path, err)
		}
		var keep []string
		for _, line := range strings.Fields(string(data)) {
			ns, err := strconv.ParseInt(line, 10, 64)
			if err == nil && now.Sub(time.Unix(0, ns)) < l.Window {
				keep = append(keep, line)
			}
		}
		if len(keep) >= l.Max {
			ok = false
		} else {
			keep = append(keep, strconv.FormatInt(now.UnixNano(), 10))
		}
		if err := f.Truncate(0); err != nil {
			return fmt.Errorf("rewrite %s: %w", l.Path, err)
		}
		if _, err := f.WriteAt([]byte(strings.Join(keep, "\n")+"\n"), 0); err != nil {
			return fmt.Errorf("rewrite %s: %w", l.Path, err)
		}
		return nil
	})
	return ok || err != nil
}

// DeliveryLog is the log of deliveries on a host, one JSON object a line,
// shared by every holder. It turns over to Path+".1" at Max bytes.
type DeliveryLog struct {
	Path string
	Lock string // the file locked while the log is written
	Max  int64
}

// DefaultLogMax is the size at which the delivery log turns over.
const DefaultLogMax = 1 << 20

// Append adds one entry.
func (l *DeliveryLog) Append(e api.AlertLogEntry) error {
	line, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode log entry: %w", err)
	}
	line = append(line, '\n')
	ctx, cancel := context.WithTimeout(context.Background(), lockWait)
	defer cancel()
	return withLock(ctx, l.Lock, func(*os.File) error {
		limit := l.Max
		if limit <= 0 {
			limit = DefaultLogMax
		}
		if st, err := os.Stat(l.Path); err == nil && st.Size()+int64(len(line)) > limit {
			if err := os.Rename(l.Path, l.Path+".1"); err != nil {
				return fmt.Errorf("turn over %s: %w", l.Path, err)
			}
		}
		f, err := os.OpenFile(l.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
		if err != nil {
			return fmt.Errorf("open %s: %w", l.Path, err)
		}
		_, err = f.Write(line)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return err
	})
}

// Tail returns the newest n entries, oldest first.
func (l *DeliveryLog) Tail(n int) ([]api.AlertLogEntry, error) {
	if n <= 0 {
		return nil, nil
	}
	var out []api.AlertLogEntry
	for _, p := range []string{l.Path, l.Path + ".1"} {
		entries, err := readLog(p)
		if err != nil {
			return nil, err
		}
		out = append(entries, out...)
		if len(out) >= n {
			break
		}
	}
	if len(out) > n {
		out = out[len(out)-n:]
	}
	return out, nil
}

func readLog(path string) ([]api.AlertLogEntry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out []api.AlertLogEntry
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		var e api.AlertLogEntry
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			out = append(out, e)
		}
	}
	return out, nil
}
