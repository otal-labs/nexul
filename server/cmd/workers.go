package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/runner"
)

// startBackgroundWorkers must run after event subscriptions, so the outbox relay never outruns consumers.
func startBackgroundWorkers(ctx context.Context, cfg *config.Config, store *storage.Store, bus *inprocess.Bus, svc *coreServices, logger *slog.Logger) (wsHandler *runner.Handler, runnerSvc *runner.Service, runnerHTTP *runner.HTTPHandler, automationsDialin *automations.DialinHandler) {
	relay := outbox.NewRelay(store.Outbox, bus, outbox.RelayConfig{Logger: logger})
	go func() {
		if err := relay.Run(ctx); err != nil {
			logger.Error("outbox relay stopped", "error", err)
		}
	}()
	// The sole occupancy feed on localhost/dev, where LiveKit's webhook can't reach back to this instance.
	go svc.voiceSvc.RunReconciliation(ctx, logger)
	// Cloudflare never pushes connector status, so a computer mid-pairing is polled and each change pushed live.
	go svc.pairingSvc.RunTunnelWatch(ctx)

	// Runners get their own connection secret, never the session-signing auth secret.
	if cfg.RunnerSecret != "" {
		if err := store.Runners.SetSecret(ctx, cfg.RunnerSecret); err != nil {
			fail(fmt.Errorf("seed runner secret: %w", err))
		}
	}
	// Published next to the DB so the compose stack's instance runner picks it up off the shared volume.
	runnerSecret, err := store.Runners.Secret(ctx)
	if err != nil {
		fail(fmt.Errorf("runner secret: %w", err))
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(cfg.DBPath), "runner-secret"), []byte(runnerSecret+"\n"), 0o600); err != nil {
		fail(fmt.Errorf("publish runner secret: %w", err))
	}
	wsHandler = runner.NewHandler(runner.HandlerConfig{
		Bus:      bus,
		Repo:     store.Runners,
		Machines: store.Machines,
		Upgrades: store.InstanceUpgrades,
		Envs:     runnerEnvLookupAdapter{deploy: svc.deploySvc},
		GitTokens: runnerGitTokenAdapter{token: func(ctx context.Context) (string, error) {
			return svc.connectorsSvc.AccessToken(ctx, "github")
		}},
		Logger: logger,
	})
	// dns can only join a connected runner's gateway once the WS handler exists (workers start after core services).
	svc.dnsSvc.SetRunnerJoin(dnsRunnerJoinAdapter{handler: wsHandler})
	releaseClient := release.New(release.Config{
		TokenSource: func(ctx context.Context) (string, error) {
			return svc.connectorsSvc.AccessToken(ctx, "github")
		},
	})
	runnerSvc = runner.NewService(store.Runners, wsHandler).WithMachines(store.Machines).WithManaged(store.Services).WithTunnelDescriber(runnerTunnelDescriberAdapter{dns: svc.dnsSvc}).WithInstall(runner.InstallConfig{
		Settings: dnsSettingsAdapter{store.Settings},
		Release:  releaseClient,
	}).WithUpgrades(store.InstanceUpgrades).WithBus(bus).WithAdminGate(instanceAdminGate{svc: svc.authSvc})
	// The update frame needs the same settings reader and release client, only available once runnerSvc exists.
	wsHandler.SetUpdateSource(dnsSettingsAdapter{store.Settings}, releaseClient)
	runnerHTTP = runner.NewHTTPHandler(runnerSvc)
	resolvePendingUpgrade(ctx, runnerSvc, logger)

	// SetConnectionRegistry closes the Service<->DialinHandler cycle so a revoked token force-drops a live connection.
	automationsDialin = automations.NewDialinHandler(svc.automationsSvc, automations.DialinConfig{
		Repo:     store.Automations,
		Cursors:  store.AutomationCursors,
		EventLog: store.AutomationEventLog,
		Runs:     store.AutomationRuns,
		Secrets:  svc.automationSecretsSvc,
		Logger:   logger,
	})
	svc.automationsSvc.SetConnectionRegistry(automationsDialin)
	go automations.RunCleanupLoop(ctx, svc.automationRunsSvc, 30*24*time.Hour, time.Hour, logger)

	return wsHandler, runnerSvc, runnerHTTP, automationsDialin
}
