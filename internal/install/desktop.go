package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// dockerDesktopHelp is the whole Windows Docker story: the installer points at Docker Desktop rather than
// installing it, because its installer needs a reboot, WSL and a licence acceptance no script can do cleanly.
const dockerDesktopHelp = `docker is not installed. Nexul runs in Docker Desktop:
install it from https://docs.docker.com/desktop/setup/install/windows-install/
(or run: winget install -e --id Docker.DockerDesktop), start it once, then run nexul install again`

// colimaResources sizes Colima's VM for the stack: OpenObserve alone may use 1 GB.
var colimaResources = []string{"--cpu", "2", "--memory", "4", "--disk", "30"}

// ensureDockerMac uses whatever Docker engine is already here, starting it when it is stopped, and otherwise
// installs Colima, a headless Docker engine, with Homebrew.
func (h *Host) ensureDockerMac(ctx context.Context) (string, error) {
	if _, err := h.LookPath("docker"); err == nil {
		if _, err := h.Exec.Run(ctx, "docker", "info"); err == nil {
			return h.dockerVersion(ctx)
		}
		return h.startDockerMac(ctx)
	}
	brew, err := h.ensureBrew(ctx)
	if err != nil {
		return "", err
	}
	if _, err := h.Exec.Run(ctx, brew, "install", "colima", "docker", "docker-compose"); err != nil {
		return "", fmt.Errorf("install Colima and Docker with Homebrew: %w", err)
	}
	if err := h.linkComposePlugin(ctx, brew); err != nil {
		return "", err
	}
	if _, err := h.Exec.Run(ctx, "colima", append([]string{"start"}, colimaResources...)...); err != nil {
		return "", fmt.Errorf("start Colima: %w", err)
	}
	// Handing the running VM to brew services is what starts it again at login.
	if _, err := h.Exec.Run(ctx, "colima", "stop"); err != nil {
		return "", fmt.Errorf("stop Colima: %w", err)
	}
	if err := h.startColimaService(ctx, brew); err != nil {
		return "", err
	}
	v, err := h.dockerVersion(ctx)
	return "installed Colima, Docker " + v, err
}

// startDockerMac starts the engine behind an installed but stopped docker: Colima, else Docker Desktop.
func (h *Host) startDockerMac(ctx context.Context) (string, error) {
	if _, err := h.LookPath("colima"); err == nil {
		brew, err := h.ensureBrew(ctx)
		if err != nil {
			return "", err
		}
		if err := h.startColimaService(ctx, brew); err != nil {
			return "", err
		}
		v, err := h.dockerVersion(ctx)
		return "started Colima, Docker " + v, err
	}
	if _, err := os.Stat(h.Paths.DockerApp); err == nil {
		if _, err := h.Exec.Run(ctx, "open", "-a", h.Paths.DockerApp); err != nil {
			return "", fmt.Errorf("start Docker Desktop: %w", err)
		}
		if err := h.waitDocker(ctx); err != nil {
			return "", err
		}
		v, err := h.dockerVersion(ctx)
		return "started Docker Desktop " + v, err
	}
	return "", errors.New("docker is installed but not running; start your Docker engine, then run nexul install again")
}

func (h *Host) startColimaService(ctx context.Context, brew string) error {
	if _, err := h.Exec.Run(ctx, brew, "services", "start", "colima"); err != nil {
		return fmt.Errorf("start Colima at login: %w", err)
	}
	return h.waitDocker(ctx)
}

// waitDocker polls until the Docker daemon answers; an engine VM takes a minute or more to boot.
func (h *Host) waitDocker(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, h.HealthTimeout)
	defer cancel()
	for {
		if _, err := h.Exec.Run(ctx, "docker", "info"); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("docker did not start; start it by hand, then run nexul install again")
		case <-time.After(h.PollInterval):
		}
	}
}

// ensureBrew finds Homebrew, installing it when missing. Its installer asks for the user's password, so it
// needs a terminal.
func (h *Host) ensureBrew(ctx context.Context) (string, error) {
	if brew, ok := h.findBrew(); ok {
		return brew, nil
	}
	if !h.Interactive {
		return "", errors.New("docker is missing and installing it needs Homebrew; install Homebrew from https://brew.sh, or run this in a terminal")
	}
	h.printf("\n  Installing Homebrew; it asks for your password.\n")
	script := filepath.Join(os.TempDir(), "nexul-brew-install.sh")
	if err := h.download(ctx, h.BrewInstallURL, script, 0o700); err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(script) }() // a leftover temp script is harmless
	if err := h.Exec.RunAttached(ctx, "/bin/bash", script); err != nil {
		return "", fmt.Errorf("install Homebrew: %w", err)
	}
	brew, ok := h.findBrew()
	if !ok {
		return "", errors.New("brew was not found after installing Homebrew; open a new terminal and run nexul install again")
	}
	return brew, nil
}

// findBrew looks on the PATH and in Homebrew's two standard prefixes, and puts the prefix's bin on this process's
// PATH so the tools it installs are found right after.
func (h *Host) findBrew() (string, bool) {
	for _, candidate := range []string{"brew", "/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		path, err := h.LookPath(candidate)
		if err != nil {
			continue
		}
		h.PrependPath(filepath.Dir(path))
		return path, true
	}
	return "", false
}

// linkComposePlugin makes Homebrew's docker-compose answer to `docker compose`, the step its install notes ask for.
func (h *Host) linkComposePlugin(ctx context.Context, brew string) error {
	prefix, err := h.Exec.Run(ctx, brew, "--prefix", "docker-compose")
	if err != nil {
		return fmt.Errorf("find docker-compose: %w", err)
	}
	if err := os.MkdirAll(h.Paths.UserPlugins, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", h.Paths.UserPlugins, err)
	}
	link := filepath.Join(h.Paths.UserPlugins, "docker-compose")
	if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace %s: %w", link, err)
	}
	if err := os.Symlink(filepath.Join(strings.TrimSpace(prefix), "bin", "docker-compose"), link); err != nil {
		return fmt.Errorf("link the compose plugin: %w", err)
	}
	return nil
}

// ensureComposeMac installs Homebrew's compose plugin when `docker compose` is missing next to an engine that
// came from somewhere else.
func (h *Host) ensureComposeMac(ctx context.Context) (string, error) {
	if v, err := h.Exec.Run(ctx, "docker", "compose", "version", "--short"); err == nil {
		return v, nil
	}
	brew, err := h.ensureBrew(ctx)
	if err != nil {
		return "", err
	}
	if _, err := h.Exec.Run(ctx, brew, "install", "docker-compose"); err != nil {
		return "", fmt.Errorf("install docker-compose with Homebrew: %w", err)
	}
	if err := h.linkComposePlugin(ctx, brew); err != nil {
		return "", err
	}
	v, err := h.Exec.Run(ctx, "docker", "compose", "version", "--short")
	if err != nil {
		return "", fmt.Errorf("compose plugin installed but not usable: %w", err)
	}
	return "installed " + v, nil
}

// ensureDockerWindows only checks: it says what to install or start instead of installing Docker Desktop itself.
func (h *Host) ensureDockerWindows(ctx context.Context) (string, error) {
	if _, err := h.LookPath("docker"); err != nil {
		return "", errors.New(dockerDesktopHelp)
	}
	if _, err := h.Exec.Run(ctx, "docker", "info"); err != nil {
		return "", errors.New("the Docker engine is not running: start Docker Desktop, wait until it shows Engine running, then run nexul install again")
	}
	return h.dockerVersion(ctx)
}

func (h *Host) ensureComposeWindows(ctx context.Context) (string, error) {
	v, err := h.Exec.Run(ctx, "docker", "compose", "version", "--short")
	if err != nil {
		return "", errors.New("the compose plugin is missing: update Docker Desktop, which includes it, then run nexul install again")
	}
	return v, nil
}
