package secretref

import (
	"context"
	"errors"
	"fmt"
)

// The typed failures a resolver reports for one reference; a vault the token cannot reach is ErrNoAccess.
var (
	ErrNotFound     = errors.New("secret not found")
	ErrFieldMissing = errors.New("field missing")
	ErrNoAccess     = errors.New("no access")
	ErrUnreachable  = errors.New("secrets manager unreachable")
	// ErrFailed is any other failure for one reference.
	ErrFailed = errors.New("secret could not be resolved")
	// ErrNoResolver means no manager is configured for a reference's scheme.
	ErrNoResolver = errors.New("no resolver for scheme")
)

// Result is one reference's outcome: a value or an error, never both. Its Err never carries the value.
type Result struct {
	Value string
	Err   error
}

// Resolver resolves references of one manager's scheme in a single batch.
type Resolver interface {
	// Scheme is the Ref.Scheme it serves, without "://".
	Scheme() string
	// ResolveAll returns one Result per ref, in order. The error is for the call as a whole.
	ResolveAll(ctx context.Context, refs []string) ([]Result, error)
}

// Registry picks a resolver by a reference's scheme.
type Registry map[string]Resolver

// NewRegistry indexes resolvers by their scheme.
func NewRegistry(resolvers ...Resolver) Registry {
	r := make(Registry, len(resolvers))
	for _, res := range resolvers {
		r[res.Scheme()] = res
	}
	return r
}

// For returns the resolver for a well-formed reference, or ErrNoResolver. The value never appears in the error.
func (r Registry) For(value string) (Resolver, error) {
	kind, ref := Parse(value)
	if kind != Reference {
		return nil, fmt.Errorf("%w: not a secret reference", ErrNoResolver)
	}
	res, ok := r[ref.Scheme]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrNoResolver, ref.Scheme)
	}
	return res, nil
}
