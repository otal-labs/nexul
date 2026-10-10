package main

import (
	"context"
	"fmt"
	"net/http"
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

	if err := cfg.EnrollOnFirstStart(ctx, &http.Client{Timeout: 30 * time.Second}, version.Version); err != nil {
		fail(err)
	}
	if err := cfg.Validate(); err != nil {
		fail(err)
	}

	client := runner.NewClient(runner.ClientConfig{
		URL:        cfg.WSURL(),
		Credential: cfg.Credential,
		Name:       cfg.Name,
		Version:    version.Version,
		Logger:     logger,
		Executor: runner.NewShellExecutor(nil, runner.ExecutorConfig{
			GitToken: cfg.GitToken, StackRoot: cfg.StackRoot, Ctl: cfg.Ctl,
		}, logger),
		HeartbeatInterval: cfg.HeartbeatInterval,
		ConnectTimeout:    cfg.ConnectTimeout,
		BackoffBase:       cfg.BackoffBase,
		BackoffMax:        cfg.BackoffMax,
		Personal:          cfg.Mode == runner.ModePersonal,
		StreamURL:         cfg.StreamURL(),
		T3Home:            cfg.T3Home,
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
