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

// resolvedEnv is a request whose references hold their values.
type resolvedEnv struct {
	req DeployRequestedEvent
	// line is the deploy-log line; empty when the env held no reference.
	line string
	// mask hides the resolved values in a deploy's log and errors; nil when there are none.
	mask *strings.Replacer
	// refs is each resolved variable's reference, for the fingerprints the secret check compares against; nil when none.
	refs map[string]string
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
	var secrets []string
	for scheme, keys := range byScheme {
		values, reasons := resolveBatch(ctx, reg, scheme, req.Env, keys)
		maps.Copy(failed, reasons)
		for k, v := range values {
			env[k] = v
			secrets = append(secrets, v)
		}
	}
	if len(failed) > 0 {
		return resolvedEnv{}, resolveError(failed)
	}
	count := 0
	refs := map[string]string{}
	for _, keys := range byScheme {
		count += len(keys)
		for _, k := range keys {
			refs[k] = req.Env[k]
		}
	}
	req.Env = env
	noun := "references"
	if count == 1 {
		noun = "reference"
	}
	return resolvedEnv{
		req:  req,
		line: fmt.Sprintf("Resolved %d secret %s in %.1fs", count, noun, time.Since(start).Seconds()),
		mask: maskOf(secrets),
		refs: refs,
	}, nil
}

// maskOf hides each value and each line of a multi-line one, longest first so a value that is the start of another
// leaves none of it showing; nil when no value is non-empty.
func maskOf(values []string) *strings.Replacer {
	var olds []string
	for _, v := range values {
		olds = append(olds, v)
		if !strings.Contains(v, "\n") {
			continue
		}
		for line := range strings.SplitSeq(v, "\n") {
			olds = append(olds, strings.TrimSuffix(line, "\r"))
		}
	}
	olds = slices.DeleteFunc(olds, func(s string) bool { return s == "" })
	if len(olds) == 0 {
		return nil
	}
	slices.SortStableFunc(olds, func(a, b string) int { return len(b) - len(a) })
	pairs := make([]string, 0, 2*len(olds))
	for _, o := range slices.Compact(olds) {
		pairs = append(pairs, o, "***")
	}
	return strings.NewReplacer(pairs...)
}

// resolveBatch resolves the variables holding one scheme's references in a single call to its resolver.
func resolveBatch(ctx context.Context, reg secretref.Registry, scheme string, env map[string]string, keys []string) (map[string]string, map[string]string) {
	failed := map[string]string{}
	name := secretref.ManagerName(scheme)
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
			failed[k] = secretref.Reason(name, err)
		}
		return nil, failed
	}
	values := map[string]string{}
	for _, k := range keys {
		r := results[index[env[k]]]
		if r.Err != nil {
			failed[k] = secretref.Reason(name, r.Err)
			continue
		}
		values[k] = r.Value
	}
	return values, failed
}

func resolveError(failed map[string]string) error {
	parts := make([]string, 0, len(failed))
	for _, k := range slices.Sorted(maps.Keys(failed)) {
		parts = append(parts, k+": "+failed[k])
	}
	return errors.New("secret references could not be resolved (" + strings.Join(parts, "; ") + ")")
}

// withSecrets resolves req's references and returns the request holding the values with an executor whose deploy log
// and errors never show them. The output a caller parses (output, headCommit) stays raw. The first deploy-log line says how many were resolved. The
// returned send also keeps the stack's fingerprints once the deploy reports healthy.
func (e *ShellExecutor) withSecrets(ctx context.Context, req DeployRequestedEvent, send func(Frame), logs *logStream) (*ShellExecutor, DeployRequestedEvent, func(Frame), error) {
	var registry secretref.Registry
	if e.registry != nil {
		registry = e.registry()
	}
	got, err := resolveEnv(ctx, registry, req)
	if err != nil {
		return e, req, send, err
	}
	send = e.keepOnHealthy(send, got)
	if got.mask == nil {
		return e, req, send, nil
	}
	logs.step(LogPhaseDeploy, got.line)
	logs.flush()
	cp := *e
	inner := e.cmd
	cp.rawCmd = func(ctx context.Context, dir string, logf func(string), name string, args ...string) error {
		err := inner(ctx, dir, logf, name, args...)
		if err == nil {
			return nil
		}
		return errors.New(got.mask.Replace(err.Error()))
	}
	cp.cmd = func(ctx context.Context, dir string, logf func(string), name string, args ...string) error {
		return cp.rawCmd(ctx, dir, func(s string) { logf(got.mask.Replace(s)) }, name, args...)
	}
	return &cp, got.req, send, nil
}

// keepOnHealthy wraps send so a healthy deploy_result first records what the deploy resolved: the fingerprint of each
// referenced variable, or nothing for a stack that holds no reference now, which stops it being checked.
func (e *ShellExecutor) keepOnHealthy(send func(Frame), got resolvedEnv) func(Frame) {
	slug := got.req.StackSlug
	if e.fingerprints == nil || slug == "" {
		return send
	}
	return func(f Frame) {
		if f.Type == FrameDeployResult && f.Status == DeployStatusHealthy {
			e.keepFingerprints(slug, got)
		}
		send(f)
	}
}

func (e *ShellExecutor) keepFingerprints(slug string, got resolvedEnv) {
	if len(got.refs) == 0 {
		if err := e.fingerprints.remove(slug); err != nil {
			e.log.Warn("secret check record not removed", "slug", slug, "error", err)
		}
		return
	}
	rec := stackFingerprints{DeployID: got.req.ID, Vars: make(map[string]trackedVar, len(got.refs))}
	for name, ref := range got.refs {
		fp, err := e.fingerprints.print(got.req.Env[name])
		if err != nil {
			e.log.Warn("secret check cannot fingerprint", "slug", slug, "error", err)
			return
		}
		rec.Vars[name] = trackedVar{Ref: ref, Print: fp}
	}
	if err := e.fingerprints.save(slug, rec); err != nil {
		e.log.Warn("secret check record not kept", "slug", slug, "error", err)
	}
}
