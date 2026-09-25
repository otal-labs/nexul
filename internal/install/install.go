// Package install puts a Nexul instance on a Linux host and keeps it current: the nexul CLI's install, update,
// status and uninstall commands.
package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/version"
)

// ErrUnknownCommand reports a command name the installer does not own.
var ErrUnknownCommand = errors.New("unknown command")

// DefaultDir is where an install keeps its compose file, .env, data, logs and stack checkouts.
const DefaultDir = "/data/nexul"

const (
	defaultPort     = 80
	defaultLogsPort = 5080
	logsEmail       = "nexul@nexul.local"
)

// Run executes one installer command against the real host.
func Run(ctx context.Context, cmd string, args []string) error {
	h := NewHost()
	switch cmd {
	case "install":
		return h.runInstall(ctx, args)
	case "upgrade":
		return h.runUpgrade(ctx, args)
	case "status":
		return h.Status(ctx)
	case "uninstall":
		return h.runUninstall(ctx, args)
	}
	return fmt.Errorf("%s: %w", cmd, ErrUnknownCommand)
}

// Options are the choices an install is made with; zero values fall back to the existing install, then defaults.
type Options struct {
	Dir      string
	Port     int
	LogsPort int
	Version  string
	Yes      bool
}

func (h *Host) runInstall(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	var o Options
	fs.StringVar(&o.Dir, "dir", "", "install directory (default "+DefaultDir+")")
	fs.IntVar(&o.Port, "port", 0, "port the web UI and API listen on (default 80)")
	fs.IntVar(&o.LogsPort, "logs-port", 0, "port the logs UI listens on (default 5080)")
	fs.StringVar(&o.Version, "version", "", "release to install (default: this binary's version)")
	fs.BoolVar(&o.Yes, "yes", false, "accept the defaults without asking")
	fs.BoolVar(&o.Yes, "y", false, "shorthand for --yes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return h.Install(ctx, o)
}

// Install brings the host to a running instance: Docker, Compose, git, the stack and the instance runner. Re-running
// it keeps the existing secrets and choices, so it doubles as a repair.
func (h *Host) Install(ctx context.Context, o Options) error {
	if err := h.requireRoot(); err != nil {
		return err
	}
	tag, err := installTag(o.Version)
	if err != nil {
		return err
	}
	prev, err := h.loadInstall()
	if err != nil {
		return err
	}

	h.printf("Nexul %s installer\n\n", tag)
	if o, prev, err = h.choose(o, prev); err != nil {
		return err
	}
	if err := h.checkPorts(o, prev); err != nil {
		return err
	}

	env, err := h.install(ctx, o, prev, tag)
	if err != nil {
		return err
	}
	h.printSummary(o, tag, env)
	return nil
}

// choose settles the directory, then the ports, asking for each when there is a terminal to ask on. The directory
// comes first because an install already in it supplies the port defaults.
func (h *Host) choose(o Options, prev *installed) (Options, *installed, error) {
	interactive := h.Interactive && !o.Yes
	if interactive {
		h.printf("Press Enter to accept the default.\n\n")
	}
	if o.Dir == "" && prev != nil {
		o.Dir = prev.Dir
	}
	if o.Dir == "" {
		o.Dir = DefaultDir
	}
	var err error
	if interactive {
		if o.Dir, err = h.prompt("Install directory", o.Dir); err != nil {
			return o, nil, err
		}
	}
	// A directory kept by uninstall still holds its .env, whose logs credentials OpenObserve has already adopted.
	if prev, err = installAt(o.Dir, prev); err != nil {
		return o, nil, err
	}
	o = withDefaults(o, prev)
	if !interactive {
		return o, prev, nil
	}
	if o, err = h.askPorts(o, prev); err != nil {
		return o, nil, err
	}
	h.printf("\n")
	return o, prev, nil
}

// install runs the steps in order and returns the .env it wrote; the first failing step stops the rest.
func (h *Host) install(ctx context.Context, o Options, prev *installed, tag string) (map[string]string, error) {
	if err := h.step("Docker", func() (string, error) { return h.ensureDocker(ctx) }); err != nil {
		return nil, err
	}
	if err := h.step("Docker Compose", func() (string, error) { return h.ensureCompose(ctx) }); err != nil {
		return nil, err
	}
	if err := h.step("Git", func() (string, error) { return h.ensurePackage(ctx, "git") }); err != nil {
		return nil, err
	}
	var env map[string]string
	if err := h.step("Files", func() (string, error) {
		var err error
		env, err = h.writeStack(o, prev, tag)
		return o.Dir, err
	}); err != nil {
		return nil, err
	}
	if err := h.startStack(ctx, o); err != nil {
		return nil, err
	}
	if err := h.step("Runner", func() (string, error) { return h.installRunner(ctx, o, tag) }); err != nil {
		return nil, err
	}
	if err := h.step("Command", func() (string, error) { return h.installSelf() }); err != nil {
		return nil, err
	}
	return env, nil
}

// startStack pulls the images and brings the stack up, then waits until the server answers.
func (h *Host) startStack(ctx context.Context, o Options) error {
	// Release tags never move, so an image already on the host is the right one.
	if err := h.step("Images", func() (string, error) {
		return "pulled", h.compose(ctx, o.Dir, "pull", "--quiet", "--policy", "missing")
	}); err != nil {
		return err
	}
	return h.step("Stack", func() (string, error) {
		if err := h.compose(ctx, o.Dir, "up", "--detach", "--remove-orphans"); err != nil {
			return "", err
		}
		return "running", h.waitHealthy(ctx, o.Port)
	})
}

// installTag resolves the release an install targets: the flag, else this binary's own version.
func installTag(flagVersion string) (string, error) {
	if flagVersion != "" {
		return "v" + strings.TrimPrefix(flagVersion, "v"), nil
	}
	if !version.IsRelease() {
		return "", errors.New("this is a development build, which has no published images; pass --version <release>")
	}
	return version.Version, nil
}

// withDefaults fills unset ports from the existing install, then from the defaults.
func withDefaults(o Options, prev *installed) Options {
	if o.Port == 0 {
		o.Port = prev.intEnv("NEXUL_PORT", defaultPort)
	}
	if o.LogsPort == 0 {
		o.LogsPort = prev.intEnv("NEXUL_LOGS_PORT", defaultLogsPort)
	}
	return o
}

// step prints one aligned progress line: the label, then the step's result or its failure.
func (h *Host) step(label string, fn func() (string, error)) error {
	h.printf("  %s %s ", label, strings.Repeat(".", max(2, 16-len(label))))
	detail, err := fn()
	if err != nil {
		h.printf("failed\n")
		return fmt.Errorf("%s: %w", strings.ToLower(label), err)
	}
	h.printf("%s\n", detail)
	return nil
}

func (h *Host) printSummary(o Options, tag string, env map[string]string) {
	host := h.PublicIP()
	lines := []string{
		"",
		fmt.Sprintf("Nexul %s is running.", tag),
		fmt.Sprintf("  Setup wizard   %s", siteURL(host, o.Port)),
		fmt.Sprintf("  Data           %s  (back this folder up)", o.Dir),
		fmt.Sprintf("  Logs UI        %s  user %s", siteURL(host, o.LogsPort), logsEmail),
		fmt.Sprintf("  Logs password  %s  (also in %s/.env)", env["NEXUL_LOGS_PASSWORD"], o.Dir),
		"  Upgrade        nexul upgrade",
		"  Status         nexul status",
		"Next: point a domain at this server and put HTTPS in front of it, then enter the",
		"https:// address as the instance URL in the setup wizard.",
	}
	h.printf("%s\n", strings.Join(lines, "\n"))
}

func siteURL(host string, port int) string {
	if port == 80 {
		return "http://" + host + "/"
	}
	return fmt.Sprintf("http://%s:%d/", host, port)
}
