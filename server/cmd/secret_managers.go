package main

import (
	"context"
	"log/slog"
	"sync"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/onepassword"
	"github.com/otal-labs/nexul/internal/platform/version"
)

// serverManagerSetups is one setup per secrets manager the server holds a token for.
func serverManagerSetups(cfg *config.Config) []secretref.Setup {
	if cfg.OnePasswordToken == "" {
		return nil
	}
	return []secretref.Setup{{
		Scheme: "op",
		Open: func(ctx context.Context) (secretref.Resolver, error) {
			res, err := onepassword.New(ctx, cfg.OnePasswordToken, version.Version)
			if err != nil {
				return nil, err
			}
			return res, nil
		},
	}}
}

// startSecretManagers opens the managers off the start path: creating the 1Password client costs seconds of CPU, and a
// rejected token or an outage must never keep the server from serving. Once they opened, and again whenever a manager's
// state changes, warm resolves the stored references. Retries of an unreachable manager end with ctx.
func startSecretManagers(ctx context.Context, managers *secretref.Managers, warm func(context.Context)) {
	go func() {
		var wg sync.WaitGroup
		defer wg.Wait()
		managers.Start(ctx, &wg)
		for {
			changed := managers.Changed()
			warm(ctx)
			select {
			case <-changed:
			case <-ctx.Done():
				return
			}
		}
	}()
}

// warmConnectorSecrets is the warm step for startSecretManagers: it logs each connector whose references did not resolve.
func warmConnectorSecrets(svc *connectors.Service, logger *slog.Logger) func(context.Context) {
	return func(ctx context.Context) {
		for id, p := range svc.Warm(ctx) {
			logger.Warn("connector secret not resolved", "connector", id, "manager", p.Manager, "state", p.State, "error", p.Error)
		}
	}
}
