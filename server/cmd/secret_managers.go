package main

import (
	"context"
	"sync"

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
// rejected token or an outage must never keep the server from serving. Retries of an unreachable manager end with ctx.
func startSecretManagers(ctx context.Context, managers *secretref.Managers) {
	var wg sync.WaitGroup
	go func() {
		managers.Start(ctx, &wg)
		wg.Wait()
	}()
}
