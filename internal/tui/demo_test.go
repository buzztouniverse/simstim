package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/iAmLakshya/simstim/internal/client"
)

var _ Backend = (*demoBackend)(nil)

func TestDemoBackend(t *testing.T) {
	ctx := context.Background()
	be := newDemoBackend(false)
	hosts, _ := be.LoadHosts()
	if len(hosts) != 3 {
		t.Fatalf("got %d hosts, want 3", len(hosts))
	}
	var ue *client.UserError
	if _, err := be.Dial(ctx, client.Host{Name: "hetzner-box"}); !errors.As(err, &ue) {
		t.Fatalf("the offline host dialled: %v", err)
	}
	c, err := be.Dial(ctx, client.Host{Name: "homelab"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	list, _ := c.Sessions(ctx)
	if len(list) != 4 {
		t.Fatalf("got %d sessions, want 4", len(list))
	}
	if screen, _, _, _ := c.Peek(ctx, list[0].ID, 0); len(screen) == 0 {
		t.Fatal("no preview screen")
	}
	if err := c.RenameSession(ctx, list[0].ID, "renamed"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Delete(ctx, list[1].ID); err != nil {
		t.Fatal(err)
	}
	s, _ := c.NewSession(ctx, client.NewArgs{Agent: "claude", Cwd: "~/src/app"})
	list, _ = c.Sessions(ctx)
	if len(list) != 4 || list[0].Name != "renamed" || s.Name != "app" {
		t.Fatalf("after rename, delete and new: %+v", list)
	}
	if hosts, _ := newDemoBackend(true).LoadHosts(); len(hosts) != 0 {
		t.Fatal("the empty demo has hosts")
	}
}
