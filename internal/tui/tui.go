// Package tui is the home screen: hosts, sessions, dialogs and the command palette.
package tui

import (
	"context"

	"github.com/iAmLakshya/simstim/internal/brand"
	"github.com/iAmLakshya/simstim/internal/client"
)

// Run opens the home screen.
func Run(ctx context.Context) error {
	return &client.UserError{Msg: "the home screen is not built yet", Hint: brand.Name + " ls, " + brand.Name + " new <host>, " + brand.Name + " attach <host>/<session>"}
}
