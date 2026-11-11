package runner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/secretref"
)

// managerNames is how a failure names the manager a scheme belongs to.
var managerNames = map[string]string{"op": "1Password"}

// resolvedEnv is a request whose references hold their values.
type resolvedEnv struct {
	req DeployRequestedEvent
	// line is the deploy-log line; empty when the env held no reference.
	line string
	// mask hides the resolved values wherever a command might echo them; nil when there are none.
	mask *strings.Replacer
}

// resolveEnv replaces every secret reference in req.Env with its value, one batch per manager. The result owns its
// own Env, so the caller's map keeps the references. A failure is an error naming each failing variable and a plain
// reason, never a reference or a value.
func resolveEnv(ctx context.Context, reg secretref.Registry, req DeployRequestedEvent) (resolvedEnv, error) {
	byScheme := map[string][]string{}
	failed := map[string]string{}
	for _, k := range slices.Sorted(maps.Keys(req.Env)) {
		kind, ref := secretref.Parse(req.Env[k])
		if kind == secretref.Malformed {
			failed[k] = "not a valid secret reference"
		}
		if kind == secretref.Reference {
			byScheme[ref.Scheme] = append(byScheme[ref.Scheme], k)
		}
	}
	if len(byScheme) == 0 && len(failed) == 0 {
		return resolvedEnv{req: req}, nil
	}

	start := time.Now()
	env := maps.Clone(req.Env)
	var pairs []string
	for scheme, keys := range byScheme {
		values, reasons := resolveBatch(ctx, reg, scheme, req.Env, keys)
		maps.Copy(failed, reasons)
		for k, v := range values {
			env[k] = v
			if v != "" {
				pairs = append(pairs, v, "***")
			}
		}
	}
	if len(failed) > 0 {
		return resolvedEnv{}, resolveError(failed)
	}
	count := 0
	for _, keys := range byScheme {
		count += len(keys)
	}
	req.Env = env
	noun := "references"
	if count == 1 {
		noun = "reference"
	}
	return resolvedEnv{
		req:  req,
		line: fmt.Sprintf("Resolved %d secret %s in %.1fs", count, noun, time.Since(start).Seconds()),
		mask: strings.NewReplacer(pairs...),
	}, nil
}

// resolveBatch resolves the variables holding one scheme's references in a single call to its resolver.
func resolveBatch(ctx context.Context, reg secretref.Registry, scheme string, env map[string]string, keys []string) (map[string]string, map[string]string) {
	failed := map[string]string{}
	name := managerNames[scheme]
	res, ok := reg[scheme]
	if !ok {
		for _, k := range keys {
			failed[k] = "no " + name + " token on this runner"
		}
		return nil, failed
	}
	refs := make([]string, 0, len(keys))
	index := map[string]int{}
	for _, k := range keys {
		if _, dup := index[env[k]]; !dup {
			index[env[k]] = len(refs)
			refs = append(refs, env[k])
		}
	}
	results, err := res.ResolveAll(ctx, refs)
	if err == nil && len(results) != len(refs) {
		err = secretref.ErrFailed
	}
	if err != nil {
		for _, k := range keys {
			failed[k] = reason(name, err)
		}
		return nil, failed
	}
	values := map[string]string{}
	for _, k := range keys {
		r := results[index[env[k]]]
		if r.Err != nil {
			failed[k] = reason(name, r.Err)
			continue
		}
		values[k] = r.Value
	}
	return values, failed
}

// reason is the plain words for one failure; it never carries the error's own text, which could name a reference.
func reason(manager string, err error) string {
	switch {
	case errors.Is(err, secretref.ErrFieldMissing):
		return "field not found"
	case errors.Is(err, secretref.ErrNotFound):
		return "not found"
	case errors.Is(err, secretref.ErrNoAccess):
		return "no access"
	case errors.Is(err, secretref.ErrUnreachable):
		return manager + " unreachable"
	}
	return "could not be resolved"
}

func resolveError(failed map[string]string) error {
	parts := make([]string, 0, len(failed))
	for _, k := range slices.Sorted(maps.Keys(failed)) {
		parts = append(parts, k+": "+failed[k])
	}
	return errors.New("secret references could not be resolved (" + strings.Join(parts, "; ") + ")")
}

// withSecrets resolves req's references and returns the request holding the values with an executor whose commands
// never echo them, in their output or their errors. The first deploy-log line says how many were resolved.
func (e *ShellExecutor) withSecrets(ctx context.Context, req DeployRequestedEvent, logs *logStream) (*ShellExecutor, DeployRequestedEvent, error) {
	var registry secretref.Registry
	if e.registry != nil {
		registry = e.registry()
	}
	got, err := resolveEnv(ctx, registry, req)
	if err != nil {
		return e, req, err
	}
	if got.mask == nil {
		return e, req, nil
	}
	logs.step(LogPhaseDeploy, got.line)
	logs.flush()
	cp := *e
	inner := e.cmd
	cp.cmd = func(ctx context.Context, dir string, logf func(string), name string, args ...string) error {
		err := inner(ctx, dir, func(s string) { logf(got.mask.Replace(s)) }, name, args...)
		if err == nil {
			return nil
		}
		return errors.New(got.mask.Replace(err.Error()))
	}
	return &cp, got.req, nil
}
