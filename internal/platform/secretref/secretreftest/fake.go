// Package secretreftest is the in-memory secretref.Resolver tests use instead of their own.
package secretreftest

import (
	"context"
	"slices"
	"sync"

	"github.com/otal-labs/nexul/internal/platform/secretref"
)

// Fake answers from Errs, then Values; a reference in neither is secretref.ErrNotFound.
type Fake struct {
	// SchemeName defaults to "op".
	SchemeName string
	Values     map[string]string
	// Errs gives a reference any typed error, taking precedence over Values.
	Errs map[string]error
	// CallErr fails the whole call before any reference is read.
	CallErr error

	mu    sync.Mutex
	calls [][]string
}

func (f *Fake) Scheme() string {
	if f.SchemeName == "" {
		return "op"
	}
	return f.SchemeName
}

func (f *Fake) ResolveAll(_ context.Context, refs []string) ([]secretref.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, slices.Clone(refs))
	f.mu.Unlock()
	if f.CallErr != nil {
		return nil, f.CallErr
	}
	out := make([]secretref.Result, len(refs))
	for i, ref := range refs {
		if err, ok := f.Errs[ref]; ok {
			out[i] = secretref.Result{Err: err}
			continue
		}
		value, ok := f.Values[ref]
		if !ok {
			out[i] = secretref.Result{Err: secretref.ErrNotFound}
			continue
		}
		out[i] = secretref.Result{Value: value}
	}
	return out, nil
}

// Calls is the refs of each ResolveAll call so far.
func (f *Fake) Calls() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}
