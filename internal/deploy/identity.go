package deploy

import (
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
