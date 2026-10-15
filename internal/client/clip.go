package client

import (
	"context"
	"os/exec"
	"runtime"
	"time"

	"golang.design/x/clipboard"
)

// MaxImage is the largest image pasted from the clipboard.
const MaxImage = 16 << 20

// ClipboardImage returns the PNG on the local clipboard, or nil if there is none.
func ClipboardImage() []byte {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := clipboard.Init(); err == nil {
		if b, err := clipboard.Read(ctx, clipboard.FmtImage); err == nil && len(b) > 0 && len(b) <= MaxImage {
			return b
		}
		if runtime.GOOS == "darwin" {
			return nil
		}
	}
	// Fallbacks for Linux desktops where the library cannot connect.
	for _, argv := range [][]string{
		{"wl-paste", "--no-newline", "--type", "image/png"},
		{"xclip", "-selection", "clipboard", "-t", "image/png", "-o"},
	} {
		if _, err := exec.LookPath(argv[0]); err != nil {
			continue
		}
		out, err := exec.CommandContext(ctx, argv[0], argv[1:]...).Output()
		if err == nil && len(out) > 8 && string(out[1:4]) == "PNG" && len(out) <= MaxImage {
			return out
		}
	}
	return nil
}
