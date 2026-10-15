package client

import (
	"context"

	"github.com/iAmLakshya/simstim/internal/api"
)

// AgentCheck is what a host knows about one agent.
type AgentCheck = api.AgentCheck

// AgentChecks asks the host which agents are installed, whether a login shell
// finds them, and whether they are logged in.
func (c *Conn) AgentChecks(ctx context.Context) ([]AgentCheck, error) {
	var out []AgentCheck
	err := c.Call(ctx, "agents.check", nil, &out)
	return out, err
}
