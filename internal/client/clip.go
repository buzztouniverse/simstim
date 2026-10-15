package client

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.design/x/clipboard"

	"github.com/iamlakshya/simstim/internal/brand"
)

// MaxImage is the largest image pasted from the clipboard.
const MaxImage = 16 << 20

// ClipboardImage returns the PNG on the local clipboard, or nil if there is none.
//
// For end-to-end tests, <NAME>_TEST_CLIPBOARD_IMAGE names a file to use as the
// clipboard's image instead, so a test never touches the user's real clipboard.
func ClipboardImage() []byte {
	if p := os.Getenv(brand.Env("TEST_CLIPBOARD_IMAGE")); p != "" {
		if b, err := os.ReadFile(p); err == nil && len(b) <= MaxImage {
			return b
		}
		return nil
	}
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
