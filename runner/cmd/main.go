package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/version"
	"github.com/otal-labs/nexul/internal/runner"
)

func main() {
	cfg, err := runner.LoadRunnerConfig()
	if err != nil {
		fail(err)
	}
	logger := logging.New(cfg.LogLevel)
	logger.Info("runner version", "version", version.Version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Token == "" && cfg.SecretFile != "" {
		logger.Info("waiting for the server to publish the runner secret", "path", cfg.SecretFile)
		waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		cfg.Token, err = runner.ReadSecretFile(waitCtx, cfg.SecretFile)
		cancel()
		if err != nil {
			fail(err)
		}
	}
	if err := cfg.Validate(); err != nil {
		fail(err)
	}

	client := runner.NewClient(runner.ClientConfig{
		URL:               cfg.ServerURL,
		Token:             cfg.Token,
		RunnerID:          cfg.RunnerID,
		Name:              cfg.Name,
		Machine:           cfg.Machine,
		StackRoot:         cfg.StackRoot,
		Version:           version.Version,
		Logger:            logger,
		Executor:          runner.NewShellExecutor(nil, cfg.GitToken, logger),
		HeartbeatInterval: cfg.HeartbeatInterval,
		ConnectTimeout:    cfg.ConnectTimeout,
		BackoffBase:       cfg.BackoffBase,
		BackoffMax:        cfg.BackoffMax,
	})
	if err := client.Run(ctx); err != nil {
		logger.Error("runner exited with error", "error", err)
		os.Exit(1)
	}
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, "runner:", err) // best-effort diagnostic; exit code carries the real result
	os.Exit(1)
}
