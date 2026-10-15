package cli

import (
	"context"

	"github.com/iamlakshya/simstim/internal/onboard"
	"github.com/iamlakshya/simstim/internal/tui"
)

// runHome opens the launcher.
func runHome(ctx context.Context) error {
	return tui.RunWith(ctx, tui.Options{Local: onboard.System{Providers: providers}})
}
