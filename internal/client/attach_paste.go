package client

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iAmLakshya/simstim/internal/keys"
	"github.com/iAmLakshya/simstim/internal/proto"
)

// sendUpload uploads a file and pastes its remote path into the session.
func (a *attacher) sendUpload(name string, data []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	path, err := a.conn.Upload(ctx, a.o.Session, name, data)
	if err != nil {
		a.notify("upload failed: " + err.Error())
		return
	}
	a.pasteText([]byte(path + " "))
}

// pasteText sends text the way a paste would arrive.
func (a *attacher) pasteText(t []byte) {
	if a.tracker.M.Private[2004] {
		t = append(append([]byte("\x1b[200~"), t...), []byte("\x1b[201~")...)
	}
	_ = a.conn.Send(proto.TInput, t)
}

// paste forwards a bracketed paste. When it is only local file paths (a file
// dropped on the terminal), the files are uploaded and the paths rewritten.
func (a *attacher) paste(p keys.Piece) {
	files := localFiles(p.Text)
	if len(files) == 0 {
		_ = a.conn.Send(proto.TInput, p.Raw)
		return
	}
	var remote []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil || len(data) > MaxImage {
			_ = a.conn.Send(proto.TInput, p.Raw)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		path, err := a.conn.Upload(ctx, a.o.Session, filepath.Base(f), data)
		cancel()
		if err != nil {
			a.notify("upload failed: " + err.Error())
			_ = a.conn.Send(proto.TInput, p.Raw)
			return
		}
		remote = append(remote, shellQuote(path))
	}
	a.pasteText([]byte(strings.Join(remote, " ") + " "))
}

// localFiles returns the paths in t if every token is an existing local regular file.
func localFiles(t []byte) []string {
	toks := splitShellWords(strings.TrimSpace(string(t)))
	if len(toks) == 0 || len(toks) > 20 {
		return nil
	}
	for _, s := range toks {
		if !filepath.IsAbs(s) {
			return nil
		}
		st, err := os.Stat(s)
		if err != nil || !st.Mode().IsRegular() {
			return nil
		}
	}
	return toks
}

// splitShellWords splits on unescaped spaces, handling backslash escapes and quotes,
// the way terminals quote dropped paths.
func splitShellWords(s string) []string {
	var out []string
	var cur bytes.Buffer
	var quote rune
	esc, have := false, false
	for _, r := range s {
		switch {
		case esc:
			cur.WriteRune(r)
			esc, have = false, true
		case r == '\\' && quote != '\'':
			esc = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, have = r, true
		case r == ' ' || r == '\n' || r == '\t':
			if have {
				out = append(out, cur.String())
				cur.Reset()
				have = false
			}
		default:
			cur.WriteRune(r)
			have = true
		}
	}
	if have {
		out = append(out, cur.String())
	}
	return out
}

func shellQuote(p string) string {
	if strings.ContainsAny(p, " '\"\\$`") {
		return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
	}
	return p
}
