package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/deploy"
)

// runnerEnvLookupAdapter adapts deploy to runner's EnvLookup seam: deploy.requested carries only redacted env keys.
type runnerEnvLookupAdapter struct {
	deploy *deploy.Service
}

func (a runnerEnvLookupAdapter) ResolveEnv(ctx context.Context, service string) (map[string]string, error) {
	svc, err := a.deploy.GetServiceByName(ctx, service)
	if err != nil {
		return nil, err
	}
	return svc.Env, nil
}
