package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	logger, runnerLog, logs := newLogger(cfg)
	if runnerLog != nil {
		defer func() {
			if err := runnerLog.Close(); err != nil {
				logger.Warn("close runner log", "error", err)
			}
		}()
	}
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
		RunnerLog:         runnerLog,
		Containers:        runner.HostContainerWatch(),
		Logs:              logs,
	})
	if err := client.Run(ctx); err != nil {
		logger.Error("runner exited with error", "error", err)
		os.Exit(1)
	}
}

// newLogger is the stderr logger. A machine's runner also queues its records for the server to relay; a personal
// one writes a log file of its own beside its credential instead and ships nothing.
func newLogger(cfg *runner.RunnerConfig) (*slog.Logger, *runner.RunnerLog, *logging.Queue) {
	if cfg.Mode != runner.ModePersonal {
		q := logging.NewQueue()
		return logging.NewWithQueue(cfg.LogLevel, q), nil, q
	}
	logger := logging.New(cfg.LogLevel)
	if cfg.CredentialFile == "" {
		return logger, nil, nil
	}
	file, err := runner.OpenRunnerLog(filepath.Dir(cfg.CredentialFile))
	if err != nil {
		logger.Warn("runner log file unavailable; its logs stay on the OS output", "error", err)
		return logger, nil, nil
	}
	return logging.NewTee(cfg.LogLevel, file), file, nil
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, "runner:", err) // best-effort diagnostic; exit code carries the real result
	os.Exit(1)
}
