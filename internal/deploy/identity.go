package deploy

import (
	"context"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

// triggeredBy leaves provenance empty when absent, never guessed.
func triggeredBy(r *http.Request) string {
	if a, ok := identity.ActorFromCtx(r.Context()); ok && a.ID != "" {
		return a.ID
	}
	return ""
}

// actorFromArgs resolves the acting identity from the call context for MCP provenance; absent, it's left empty.
func actorFromArgs(ctx context.Context) (identity.Actor, bool) {
	return identity.ActorFromCtx(ctx)
}
