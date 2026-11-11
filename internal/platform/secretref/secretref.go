// Package secretref recognises a secret reference: a value that names, in full, where a secret lives in an
// external secrets manager instead of holding it (ADR 0181).
package secretref

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Kind is what a value turned out to be.
type Kind int

const (
	// Literal is a stored value, including any URL whose scheme is not a manager's.
	Literal Kind = iota
	// Reference starts with a manager's scheme and has the shape that manager takes.
	Reference
	// Malformed starts with a manager's scheme but is not a reference of its shape.
	Malformed
)

// Ref is the parts of a reference.
type Ref struct {
	Scheme  string
	Vault   string
	Item    string
	Section string
	Field   string
}

// schemes is the closed list of managers; a later manager adds its prefix here.
var schemes = []string{"op://"}

// KnownScheme reports whether scheme, written without "://", is in the closed list.
func KnownScheme(scheme string) bool {
	return slices.Contains(schemes, scheme+"://")
}

// Parse reports whether value is a literal, a reference or a malformed reference. A reference is the whole
// value, so a scheme anywhere but the start leaves it a literal.
func Parse(value string) (Kind, Ref) {
	for _, scheme := range schemes {
		rest, ok := strings.CutPrefix(value, scheme)
		if !ok {
			continue
		}
		return parseOp(strings.TrimSuffix(scheme, "://"), rest)
	}
	return Literal, Ref{}
}

// parseOp reads vault/item/field or vault/item/section/field.
func parseOp(scheme, rest string) (Kind, Ref) {
	parts := strings.Split(rest, "/")
	if len(parts) < 3 || len(parts) > 4 {
		return Malformed, Ref{}
	}
	for _, p := range parts {
		if p == "" || p != strings.TrimSpace(p) {
			return Malformed, Ref{}
		}
	}
	ref := Ref{Scheme: scheme, Vault: parts[0], Item: parts[1], Field: parts[len(parts)-1]}
	if len(parts) == 4 {
		ref.Section = parts[2]
	}
	return Reference, ref
}

// ManagerName is how a person reads a manager's scheme, such as 1Password for op; an unknown scheme reads as itself.
func ManagerName(scheme string) string {
	if scheme == "op" {
		return "1Password"
	}
	return scheme
}

// Schemes lists, sorted and without repeats, the managers whose references the values hold.
func Schemes(values ...map[string]string) []string {
	seen := map[string]bool{}
	for _, env := range values {
		for _, v := range env {
			if kind, ref := Parse(v); kind == Reference {
				seen[ref.Scheme] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(seen))
}

// Check rejects a malformed reference as ErrInvalid naming what held it; the value never appears in the error.
func Check(name, value string) error {
	if kind, _ := Parse(value); kind == Malformed {
		return fmt.Errorf("%w: %s is not a valid secret reference; it must read op://vault/item/field or op://vault/item/section/field", apperrs.ErrInvalid, name)
	}
	return nil
}
