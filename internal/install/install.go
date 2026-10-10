// Package install puts Nexul on a Linux server, a Mac or a Windows PC as native services and keeps it current: the
// nexul command's install, upgrade, status and uninstall commands.
package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/otal-labs/nexul/internal/platform/version"
)

// ErrUnknownCommand reports a command name the installer does not own.
var ErrUnknownCommand = errors.New("unknown command")

// ExitCodeError carries the exit code of a command that already reported its own failure to the terminal.
type ExitCodeError struct{ Code int }

func (e *ExitCodeError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }

// DefaultDir is where a Linux install keeps its settings, data, logs and stack checkouts.
const DefaultDir = "/data/nexul"

const (
	defaultPort = 5123
	logsEmail   = "nexul@nexul.local"
	// setupCodeWait bounds the wait for the setup code, which never appears once the instance has an owner.
	setupCodeWait = 10 * time.Second
)

// Run executes one nexul command against the real host.
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
	case "service-host":
		if len(args) != 1 {
			return errors.New("usage: nexul service-host <unit>")
		}
		return h.serviceHost(args[0])
	}
	return fmt.Errorf("%s: %w", cmd, ErrUnknownCommand)
}

// Options are the choices a server install is made with; zero values fall back to the existing install, then
// defaults.
type Options struct {
	Dir     string
	Port    int
	Version string
	Yes     bool
}

func (h *Host) runInstall(ctx context.Context, args []string) error {
	if len(args) > 0 && (args[0] == kindRunner || args[0] == kindAutomations) {
		return h.runInstallHost(ctx, args[0], args[1:])
	}
	if len(args) > 0 && args[0] == kindComputer {
		return h.runInstallComputer(ctx, args[1:])
	}
	if len(args) > 0 && args[0] == kindServer {
		args = args[1:]
	}
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	var o Options
	fs.StringVar(&o.Dir, "dir", "", "install directory (default "+h.defaultDir()+")")
	fs.IntVar(&o.Port, "port", 0, "port the web UI and API listen on (default 5123)")
	fs.StringVar(&o.Version, "version", "", "release to install (default: this binary's version)")
	fs.BoolVar(&o.Yes, "yes", false, "accept the defaults without asking")
	fs.BoolVar(&o.Yes, "y", false, "shorthand for --yes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return h.Install(ctx, o)
}

// Install brings the host to a running instance: Docker for the runner, the server and OpenObserve as services, then
// the bundled runner and automations host. Re-running it keeps the existing secrets and choices, so it doubles as a
// repair.
func (h *Host) Install(ctx context.Context, o Options) error {
	if err := h.checkUser(); err != nil {
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
	if _, err := os.Stat(filepath.Join(o.Dir, "docker-compose.yml")); err == nil {
		return fmt.Errorf("%s holds a Docker Compose install from an earlier release; run that release's `nexul uninstall` first, then run this again", o.Dir)
	}
	if err := h.checkPort(o.Port, prev); err != nil {
		return err
	}

	settings, err := h.install(ctx, o, prev, tag)
	if err != nil {
		return err
	}
	h.printSummary(o, tag, settings, h.waitSetupCode(ctx, o.Dir))
	return nil
}

// choose settles the directory, then the port, asking for each when there is a terminal to ask on. The directory
// comes first because an install already in it supplies the port default.
func (h *Host) choose(o Options, prev *installed) (Options, *installed, error) {
	interactive := h.Interactive && !o.Yes
	if interactive {
		h.printf("Press Enter to accept the default.\n\n")
	}
	if o.Dir == "" && prev != nil {
		o.Dir = prev.Dir
	}
	if o.Dir == "" {
		o.Dir = h.defaultDir()
	}
	var err error
	if interactive {
		if o.Dir, err = h.askDir("Install directory", o.Dir); err != nil {
			return o, nil, err
		}
	}
	if o.Dir, err = installDir(o.Dir); err != nil {
		return o, nil, err
	}
	// A directory kept by uninstall still holds its .env, whose logs credentials OpenObserve has already adopted.
	if prev, err = installAt(o.Dir, prev); err != nil {
		return o, nil, err
	}
	if o.Port == 0 {
		o.Port = prev.intEnv("NEXUL_PORT", defaultPort)
	}
	if !interactive {
		return o, prev, nil
	}
	if o.Port, err = h.askPort("Web port", o.Port, prev); err != nil {
		return o, nil, err
	}
	h.printf("\n")
	return o, prev, nil
}

// askDir asks for the install directory until the answer is one installDir accepts.
func (h *Host) askDir(label, def string) (string, error) {
	for {
		answer, err := h.prompt(label, def)
		if err != nil {
			return "", err
		}
		if _, err := installDir(answer); err != nil {
			h.printf("  %s. Press Enter for %s, or type a full path.\n", err, def)
			continue
		}
		return answer, nil
	}
}

// installDir accepts an absolute path of printable characters. A key pressed while the installer downloads lands
// in the first prompt as an escape sequence, which would otherwise become a relative directory systemd rejects.
func installDir(dir string) (string, error) {
	if strings.IndexFunc(dir, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%q has control characters in it, likely a key pressed while this downloaded", dir)
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("%q is not a full path", dir)
	}
	return filepath.Clean(dir), nil
}

// install runs the steps in order and returns the settings it wrote; the first failing step stops the rest.
func (h *Host) install(ctx context.Context, o Options, prev *installed, tag string) (map[string]string, error) {
	for _, s := range h.prerequisites(ctx) {
		if err := h.step(s.label, s.run); err != nil {
			return nil, err
		}
	}
	if err := h.step("Command", h.installSelf); err != nil {
		return nil, err
	}
	if h.GOOS == "linux" {
		if err := h.step("User", func() (string, error) { return h.ensureServiceUser(ctx) }); err != nil {
			return nil, err
		}
	}
	var settings map[string]string
	if err := h.step("Files", func() (string, error) {
		var err error
		settings, err = h.writeSettings(ctx, o, prev, tag)
		return "done", err
	}); err != nil {
		return nil, err
	}
	steps := []prerequisite{
		{"Logs", func() (string, error) { return h.installLogs(ctx, o.Dir, settings) }},
		{"Server", func() (string, error) { return h.installServer(ctx, o, settings, tag) }},
		{"Runner", func() (string, error) { return h.installBundled(ctx, o, kindRunner, tag) }},
		{"Automations", func() (string, error) { return h.installBundled(ctx, o, kindAutomations, tag) }},
	}
	for _, s := range steps {
		if err := h.step(s.label, s.run); err != nil {
			return nil, err
		}
	}
	return settings, nil
}

type prerequisite struct {
	label string
	run   func() (string, error)
}

// prerequisites are what the runner needs to deploy stacks: a Linux server gets Docker, Compose and git installed;
// a Mac gets its Docker engine started or Colima installed; Windows is told what to install.
func (h *Host) prerequisites(ctx context.Context) []prerequisite {
	if h.GOOS == "darwin" {
		return []prerequisite{
			{"Docker", func() (string, error) { return h.ensureDockerMac(ctx) }},
			{"Docker Compose", func() (string, error) { return h.ensureComposeMac(ctx) }},
		}
	}
	if h.GOOS == "windows" {
		return []prerequisite{
			{"Docker", func() (string, error) { return h.ensureDockerWindows(ctx) }},
			{"Docker Compose", func() (string, error) { return h.ensureComposeWindows(ctx) }},
		}
	}
	return []prerequisite{
		{"Docker", func() (string, error) { return h.ensureDocker(ctx) }},
		{"Docker Compose", func() (string, error) { return h.ensureCompose(ctx) }},
		{"Git", func() (string, error) { return h.ensurePackage(ctx, "git") }},
	}
}

// installLogs puts OpenObserve in its unit directory, unless the pinned build is already there, and (re)starts it.
func (h *Host) installLogs(ctx context.Context, dir string, settings map[string]string) (string, error) {
	u := h.newUnit(kindLogs, "", h.OpenObserve.Version)
	prev, err := h.loadUnit(u.Name)
	if err != nil {
		return "", err
	}
	if prev == nil || prev.Version != u.Version {
		if err := os.MkdirAll(u.Dir, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", u.Dir, err)
		}
		if err := h.installOpenObserve(ctx, u.Exec); err != nil {
			return "", err
		}
	}
	u.WorkDir = filepath.Join(dir, "logs")
	u.Env = logsEnv(dir, settings)
	if h.GOOS == "linux" {
		u.User = serviceUser
	}
	if err := h.saveUnit(u); err != nil {
		return "", err
	}
	if err := h.services().install(ctx, u); err != nil {
		return "", err
	}
	return "OpenObserve " + u.Version + " on 127.0.0.1:" + settings["NEXUL_LOGS_PORT"], nil
}

// installServer puts nexul-server in its unit directory, (re)starts it and waits until it answers.
func (h *Host) installServer(ctx context.Context, o Options, settings map[string]string, tag string) (string, error) {
	u := h.newUnit(kindServer, "", tag)
	if err := os.MkdirAll(u.Dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", u.Dir, err)
	}
	if err := h.releaseAsset(ctx, tag, h.assetName(binaryFor(kindServer)), u.Exec); err != nil {
		return "", err
	}
	u.WorkDir = filepath.Join(o.Dir, "data")
	u.Env = serverEnv(o.Dir, o.Port, settings, h.desktop())
	if h.GOOS == "linux" {
		u.User = serviceUser
	}
	if err := h.saveUnit(u); err != nil {
		return "", err
	}
	if err := h.services().install(ctx, u); err != nil {
		return "", err
	}
	return "running on port " + strconv.Itoa(o.Port), h.waitHealthy(ctx, o.Port, u)
}

// serverEnv points the server at its database and at the local OpenObserve, which it proxies at /openobserve/ and
// ships its own logs to. A local install is one computer trying Nexul, where localhost is a valid instance URL.
func serverEnv(dir string, port int, settings map[string]string, local bool) map[string]string {
	logs := "http://127.0.0.1:" + settings["NEXUL_LOGS_PORT"]
	env := map[string]string{
		"NEXUL_HTTP_ADDR":     ":" + strconv.Itoa(port),
		"NEXUL_DB_PATH":       filepath.Join(dir, "data", "nexul.db"),
		"NEXUL_LOG_LEVEL":     "info",
		"NEXUL_LOGS_URL":      logs,
		"NEXUL_OTLP_ENDPOINT": logs + openObserveBase + "/api/default",
		"NEXUL_OTLP_USER":     settings["NEXUL_LOGS_EMAIL"],
		"NEXUL_OTLP_TOKEN":    settings["NEXUL_LOGS_TOKEN"],
	}
	if local {
		env["NEXUL_LOCAL"] = "1"
	}
	return env
}

// installBundled enrolls the instance's own runner or automations host, named instance, through the same path as a
// remote one, with the code the server wrote at boot. One already installed is left as it is.
func (h *Host) installBundled(ctx context.Context, o Options, kind, tag string) (string, error) {
	existing, err := h.loadUnit(unitName(kind, "instance"))
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.Name + ", already installed", nil
	}
	code, err := h.waitCode(ctx, filepath.Join(o.Dir, "data", "enroll", kind+"-instance"))
	if err != nil {
		return "", err
	}
	host := HostOptions{Kind: kind, Server: fmt.Sprintf("http://127.0.0.1:%d", o.Port), Name: "instance", Code: code}
	if kind == kindRunner {
		host.StackRoot = o.Dir
	}
	return h.installHost(ctx, host, tag)
}

// waitCode reads an enrollment code file the server writes at boot, giving it the health timeout to appear.
func (h *Host) waitCode(ctx context.Context, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, h.HealthTimeout)
	defer cancel()
	for {
		if code := readCode(path); code != "" {
			return code, nil
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("the server wrote no enrollment code to %s", path)
		case <-time.After(h.PollInterval):
		}
	}
}

// readCode returns the code in a file the server writes, or "" while there is none.
func readCode(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// setupCodePath is where the server keeps the one-time setup code while the instance has no owner.
func setupCodePath(dir string) string {
	return filepath.Join(dir, "data", "enroll", "setup")
}

// waitSetupCode gives the server a moment to write the setup code; "" means the instance already has an owner.
func (h *Host) waitSetupCode(ctx context.Context, dir string) string {
	ctx, cancel := context.WithTimeout(ctx, setupCodeWait)
	defer cancel()
	code, err := h.waitCode(ctx, setupCodePath(dir))
	if err != nil {
		return "" // the only failure is the wait running out, which is how an owned instance answers
	}
	return code
}

// installTag resolves the release an install targets: the flag, else this binary's own version.
func installTag(flagVersion string) (string, error) {
	if flagVersion != "" {
		return "v" + strings.TrimPrefix(flagVersion, "v"), nil
	}
	if !version.IsRelease() {
		return "", errors.New("this is a development build, which has no published release; pass --version <release>")
	}
	return version.Version, nil
}

// step prints one aligned progress line: the label, then the step's result or its failure.
func (h *Host) step(label string, fn func() (string, error)) error {
	line := fmt.Sprintf("  %s %s ", label, strings.Repeat(".", max(2, 16-len(label))))
	h.printf("%s", line)
	stop := h.showElapsed(line)
	detail, err := fn()
	stop()
	if err != nil {
		h.printf("failed\n")
		return fmt.Errorf("%s: %w", strings.ToLower(label), err)
	}
	h.printf("%s\n", detail)
	return nil
}

// elapsed is a running step's line and the time last drawn after it.
type elapsed struct {
	line  string
	shown string
}

// showElapsed redraws a running step's line every second with its elapsed time, so a slow step (installing Docker)
// does not look hung. Any other output during the step ends it, so an attached installer's prompt is never overwritten.
func (h *Host) showElapsed(line string) (stop func()) {
	if !h.Live {
		return func() {}
	}
	e := &elapsed{line: line}
	h.outMu.Lock()
	h.running = e
	h.outMu.Unlock()
	start := time.Now()
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case now := <-tick.C:
				h.redraw(e, now.Sub(start).Round(time.Second).String())
			}
		}
	})
	return func() {
		close(done)
		wg.Wait()
		h.redraw(e, "")
		h.outMu.Lock()
		if h.running == e {
			h.running = nil
		}
		h.outMu.Unlock()
	}
}

// redraw rewrites e's line with shown after it, blanking what was drawn before; it does nothing once e is displaced.
func (h *Host) redraw(e *elapsed, shown string) {
	h.outMu.Lock()
	defer h.outMu.Unlock()
	if h.running != e || shown == e.shown {
		return
	}
	pad := strings.Repeat(" ", max(0, len(e.shown)-len(shown)))
	_, _ = fmt.Fprintf(h.Out, "\r%s%s%s\r%s%s", e.line, shown, pad, e.line, shown) // terminal writes have nowhere to report failure
	e.shown = shown
}

func (h *Host) printSummary(o Options, tag string, settings map[string]string, code string) {
	site := siteURL(h.PublicIP(), o.Port)
	var next []string
	if code != "" {
		next = []string{
			"Next: open the setup page and enter the setup code. The domain and HTTPS, including",
			"ports 80 and 443, are set up from there with a reverse proxy or a Cloudflare tunnel,",
			"not by this installer.",
		}
	}
	if h.desktop() {
		next = []string{"This instance runs on this computer. To put Nexul on a server, run the same install there."}
	}
	lines := []string{
		"",
		fmt.Sprintf("Nexul %s is running.", tag),
		fmt.Sprintf("  Setup page     %s", site),
	}
	if code != "" {
		lines = append(lines, fmt.Sprintf("  Setup code     %s", code))
	}
	lines = append(lines,
		fmt.Sprintf("  Data           %s  (back this folder up)", o.Dir),
		fmt.Sprintf("  Logs UI        %sopenobserve/  user %s", site, logsEmail),
		fmt.Sprintf("  Logs password  %s  (also in %s)", settings["NEXUL_LOGS_PASSWORD"], filepath.Join(o.Dir, ".env")),
		"  Upgrade        nexul upgrade",
		"  Status         nexul status",
	)
	if h.GOOS == "linux" {
		lines = append(lines, fmt.Sprintf("  Firewall       port %d now accepts Docker containers on this machine, so a tunnel can reach Nexul", o.Port))
	}
	h.printf("%s\n", strings.Join(append(lines, next...), "\n"))
}

func siteURL(host string, port int) string {
	if port == 80 {
		return "http://" + host + "/"
	}
	return fmt.Sprintf("http://%s:%d/", host, port)
}
