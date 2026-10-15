package main

import (
	"context"

	"github.com/iAmLakshya/simstim/internal/tui"
)

// runHome opens the launcher.
func runHome(ctx context.Context) error {
	return tui.Run(ctx)
}
