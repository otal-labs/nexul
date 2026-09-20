package integrations

import (
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Scope is one granted capability (ADR 0043): a permissions.Action, "<domain>:<action>", action one of read/write/delete.
type Scope string

// action is one of the three capabilities a domain can grant.
type action string

const (
	actionRead   action = "read"
	actionWrite  action = "write"
	actionDelete action = "delete"
)

// Constants kept for callers outside this package (automations, storage, integration tests)
// that reference a scope by name rather than building one from a string.
const (
	ScopeEventsRead    Scope = "events:read"
	ScopeDocsRead      Scope = "docs:read"
	ScopeDocsWrite     Scope = "docs:write"
	ScopeTicketsRead   Scope = "tickets:read"
	ScopeTicketsWrite  Scope = "tickets:write"
	ScopeDeploysRead   Scope = "deploys:read"
	ScopeTopologyRead  Scope = "topology:read"
	ScopeTopologyWrite Scope = "topology:write"
	ScopeProjectsRead  Scope = "projects:read"
)

// ScopeInfo is one catalog entry, the same shape as permissions.Info so tokens and roles render one grid.
type ScopeInfo = permissions.Info

// Catalog is the shared permissions catalog: a token can be granted exactly what a role can.
func Catalog() []ScopeInfo {
	return permissions.Catalog()
}

// allScopes is the set of every grantable scope; auth and pairing have no row, so tokens never reach them.
var allScopes = buildAllScopes()

func buildAllScopes() map[Scope]bool {
	catalog := Catalog()
	out := make(map[Scope]bool, len(catalog))
	for _, info := range catalog {
		out[Scope(info.Value)] = true
	}
	return out
}

// impliedScopes maps a write or delete scope to the read scope it also requires (ADR 0010): nothing
// else is implied, so write does not imply delete and delete does not imply write.
var impliedScopes = buildImpliedScopes()

func buildImpliedScopes() map[Scope][]Scope {
	out := map[Scope][]Scope{}
	for _, info := range Catalog() {
		if info.Action == string(actionRead) {
			continue
		}
		read := Scope(info.Domain + ":" + string(actionRead))
		if !allScopes[read] {
			continue
		}
		out[Scope(info.Value)] = []Scope{read}
	}
	return out
}

// ParseScope validates a single scope string against the grantable set.
func ParseScope(s string) (Scope, error) {
	sc := Scope(strings.TrimSpace(s))
	if !allScopes[sc] {
		return "", fmt.Errorf("%w: unknown scope %q", apperrs.ErrInvalid, s)
	}
	return sc, nil
}

// ParseScopes validates a requested scope list; an empty list is invalid, least privilege still needs one capability.
func ParseScopes(raw []string) ([]Scope, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: at least one scope is required", apperrs.ErrInvalid)
	}
	out := make([]Scope, 0, len(raw))
	for _, r := range raw {
		sc, err := ParseScope(r)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(out, sc) {
			out = append(out, sc)
		}
	}
	return out, nil
}

// ResolveScopes is ParseScopes + ExpandScopes as plain strings, injected into automations without a cross-import (ADR 0017).
func ResolveScopes(raw []string) ([]string, error) {
	parsed, err := ParseScopes(raw)
	if err != nil {
		return nil, err
	}
	effective := ExpandScopes(parsed)
	out := make([]string, len(effective))
	for i, sc := range effective {
		out[i] = string(sc)
	}
	return out, nil
}

// ExpandScopes returns the requested scopes plus the reads the writes/deletes imply; this is what the token enforces (ADR 0010).
func ExpandScopes(requested []Scope) []Scope {
	effective := slices.Clone(requested)
	for _, sc := range requested {
		for _, implied := range impliedScopes[sc] {
			if !slices.Contains(effective, implied) {
				effective = append(effective, implied)
			}
		}
	}
	return effective
}

// HasScopes reports whether the scope set contains sc.
func HasScopes(scopes []Scope, sc Scope) bool {
	return slices.Contains(scopes, sc)
}
