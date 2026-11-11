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
	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/onepassword"
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
	if cfg.Mode == runner.ModePersonal {
		runner.UseLoginPath(ctx, logger)
	}

	if err := cfg.EnrollOnFirstStart(ctx, &http.Client{Timeout: 30 * time.Second}, version.Version); err != nil {
		fail(err)
	}
	if err := cfg.Validate(); err != nil {
		fail(err)
	}

	managers := openManagers(cfg, logger)
	execCfg := runner.ExecutorConfig{GitToken: cfg.GitToken, StackRoot: cfg.StackRoot, Ctl: cfg.Ctl}
	var secretCheck *runner.SecretCheck
	if cfg.Mode != runner.ModePersonal {
		execCfg.Fingerprints = runner.NewFingerprints(cfg.StateDir())
	}
	if managers != nil {
		execCfg.Registry = managers.DeployRegistry
		secretCheck = runner.NewSecretCheck(execCfg.Fingerprints, managers.Registry, cfg.SecretCheckInterval, logger)
	}
	client := runner.NewClient(runner.ClientConfig{
		URL:               cfg.WSURL(),
		Credential:        cfg.Credential,
		Name:              cfg.Name,
		Version:           version.Version,
		Logger:            logger,
		Executor:          runner.NewShellExecutor(nil, execCfg, logger),
		HeartbeatInterval: cfg.HeartbeatInterval,
		ConnectTimeout:    cfg.ConnectTimeout,
		BackoffBase:       cfg.BackoffBase,
		BackoffMax:        cfg.BackoffMax,
		Personal:          cfg.Mode == runner.ModePersonal,
		StreamURL:         cfg.StreamURL(),
		T3Home:            cfg.T3Home,
		T3InstallingFile:  cfg.T3InstallingFile(),
		T3Supervised:      cfg.T3Supervised,
		RunnerLog:         runnerLog,
		Containers:        runner.HostContainerWatch(),
		Logs:              logs,
		Managers:          managers,
		SecretCheck:       secretCheck,
	})
	if err := client.Run(ctx); err != nil {
		logger.Error("runner exited with error", "error", err)
		os.Exit(1)
	}
}

// opTokenEnv is the 1Password service account token `nexul runner 1password` puts in the runner's unit.
const opTokenEnv = "OP_SERVICE_ACCOUNT_TOKEN"

// openManagers sets up the managers this runner holds a token for. The token leaves the environment once read, so
// no container or command the runner starts inherits it; a personal runner takes none.
func openManagers(cfg *runner.RunnerConfig, logger *slog.Logger) *secretref.Managers {
	token := os.Getenv(opTokenEnv)
	if err := os.Unsetenv(opTokenEnv); err != nil {
		logger.Warn("clear the 1password token from the environment", "error", err)
	}
	if token == "" || cfg.Mode == runner.ModePersonal {
		return nil
	}
	return secretref.NewManagers(logger, secretref.Setup{
		Scheme: "op",
		Open: func(ctx context.Context) (secretref.Resolver, error) {
			res, err := onepassword.New(ctx, token, version.Version)
			if err != nil {
				return nil, err
			}
			return res, nil
		},
	})
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
