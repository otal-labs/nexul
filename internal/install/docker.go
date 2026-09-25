package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ensureDocker installs Docker Engine through Docker's own script when it is missing, and starts the daemon when it
// is installed but stopped. The script sets up Docker's package repository, so later upgrades come from the OS.
func (h *Host) ensureDocker(ctx context.Context) (string, error) {
	path, err := h.LookPath("docker")
	if err != nil {
		if err := h.installDocker(ctx); err != nil {
			return "", err
		}
		v, err := h.dockerVersion(ctx)
		return "installed " + v, err
	}
	if strings.HasPrefix(path, "/snap/") {
		return "", errors.New("docker from snap cannot bind-mount /data; remove it (snap remove docker) and run this again")
	}
	if _, err := h.Exec.Run(ctx, "docker", "info"); err != nil {
		if _, err := h.Exec.Run(ctx, "systemctl", "enable", "--now", "docker"); err != nil {
			return "", fmt.Errorf("docker is installed but its daemon is not running: %w", err)
		}
	}
	return h.dockerVersion(ctx)
}

func (h *Host) installDocker(ctx context.Context) error {
	script := filepath.Join(os.TempDir(), "nexul-get-docker.sh")
	if err := h.download(ctx, h.DockerScriptURL, script, 0o700); err != nil {
		return err
	}
	defer func() { _ = os.Remove(script) }() // a leftover temp script is harmless
	if _, err := h.Exec.Run(ctx, "sh", script); err != nil {
		return fmt.Errorf("install docker: %w", err)
	}
	return nil
}

func (h *Host) dockerVersion(ctx context.Context) (string, error) {
	return h.Exec.Run(ctx, "docker", "version", "--format", "{{.Server.Version}}")
}

// ensureCompose installs the Compose plugin when `docker compose` is missing: first the package matching the
// host's package manager, then Docker's documented manual install into the CLI plugins directory.
func (h *Host) ensureCompose(ctx context.Context) (string, error) {
	if v, err := h.Exec.Run(ctx, "docker", "compose", "version", "--short"); err == nil {
		return v, nil
	}
	if pm, ok := h.packageManager(); ok {
		for _, pkg := range composePackages[pm.name] {
			if err := h.installPackage(ctx, pm, pkg); err != nil {
				continue
			}
			if v, err := h.Exec.Run(ctx, "docker", "compose", "version", "--short"); err == nil {
				return "installed " + v + " (" + pkg + ")", nil
			}
		}
	}
	if err := h.downloadCompose(ctx); err != nil {
		return "", err
	}
	v, err := h.Exec.Run(ctx, "docker", "compose", "version", "--short")
	if err != nil {
		return "", fmt.Errorf("compose plugin installed but not usable: %w", err)
	}
	return "installed " + v, nil
}

// composePackages lists, per package manager, the packages that provide `docker compose`: Docker's own repository
// names it docker-compose-plugin, Ubuntu's docker.io pairs with docker-compose-v2.
var composePackages = map[string][]string{
	"apt-get": {"docker-compose-plugin", "docker-compose-v2"},
	"dnf":     {"docker-compose-plugin"},
	"yum":     {"docker-compose-plugin"},
	"zypper":  {"docker-compose"},
	"pacman":  {"docker-compose"},
	"apk":     {"docker-cli-compose"},
}

func (h *Host) downloadCompose(ctx context.Context) error {
	arch, ok := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[h.GOARCH]
	if !ok {
		return fmt.Errorf("no compose plugin build for %s; install docker compose, then run this again", h.GOARCH)
	}
	name := "docker-compose-linux-" + arch
	sum, err := h.fetchChecksum(ctx, h.ComposeURL+"/"+name+".sha256", name)
	if err != nil {
		return err
	}
	return h.downloadVerified(ctx, h.ComposeURL+"/"+name, filepath.Join(h.Paths.ComposePlugs, "docker-compose"), sum)
}

// ensurePackage installs a command's package through the host's package manager when the command is missing.
func (h *Host) ensurePackage(ctx context.Context, name string) (string, error) {
	if _, err := h.LookPath(name); err == nil {
		return "present", nil
	}
	pm, ok := h.packageManager()
	if !ok {
		return "", fmt.Errorf("%s is missing and no supported package manager was found; install it, then run this again", name)
	}
	if err := h.installPackage(ctx, pm, name); err != nil {
		return "", err
	}
	return "installed", nil
}

type packageManager struct {
	name    string
	install []string
}

var packageManagers = []packageManager{
	{"apt-get", []string{"install", "-y", "-qq"}},
	{"dnf", []string{"install", "-y", "-q"}},
	{"yum", []string{"install", "-y", "-q"}},
	{"zypper", []string{"--non-interactive", "install"}},
	{"pacman", []string{"-S", "--noconfirm", "--needed"}},
	{"apk", []string{"add", "--no-cache"}},
}

func (h *Host) packageManager() (packageManager, bool) {
	for _, pm := range packageManagers {
		if _, err := h.LookPath(pm.name); err == nil {
			return pm, true
		}
	}
	return packageManager{}, false
}

func (h *Host) installPackage(ctx context.Context, pm packageManager, pkg string) error {
	if pm.name == "apt-get" && !h.aptUpdated {
		if _, err := h.Exec.Run(ctx, "apt-get", "update", "-qq"); err != nil {
			return err
		}
		h.aptUpdated = true
	}
	_, err := h.Exec.Run(ctx, pm.name, slices.Concat(pm.install, []string{pkg})...)
	return err
}
