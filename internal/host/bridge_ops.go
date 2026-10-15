package host

import (
	"github.com/iAmLakshya/simstim/internal/proto"
)

// moreFeatures are the features of the requests in this file.
var moreFeatures = []string{"agents.check"}

func init() { Features = append(Features, moreFeatures...) }

// handleMore serves the requests that set a host up. It reports whether it
// knew the request.
func (b *bridge) handleMore(req proto.Request) bool {
	switch req.Op {
	case "agents.check":
		b.reply(req.ID, AgentChecks(), nil)
	default:
		return false
	}
	return true
}
