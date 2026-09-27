package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HostOptions are the choices a runner or automations host is installed with.
type HostOptions struct {
	Kind      string
	Server    string
	Name      string
	Code      string
	StackRoot string
	GitToken  string
	Version   string
	Yes       bool
}

func (h *Host) runInstallHost(ctx context.Context, kind string, args []string) error {
	fs := flag.NewFlagSet("install "+kind, flag.ContinueOnError)
	fs.SetOutput(h.Out)
	o := HostOptions{Kind: kind}
	fs.StringVar(&o.Server, "server", "", "the instance's address, e.g. https://nexul.example.com")
	fs.StringVar(&o.Name, "name", "", "this "+kind+"'s name: lower case letters, digits and dashes")
	fs.StringVar(&o.Code, "code", "", "the one-time enrollment code the instance gave you")
	if kind == kindRunner {
		fs.StringVar(&o.StackRoot, "stack-root", "", "the stack root; checkouts go in its stacks folder (default: the runner's directory)")
		fs.StringVar(&o.GitToken, "git-token", "", "a token for cloning private repositories")
	}
	fs.StringVar(&o.Version, "version", "", "release to install (default: this binary's version)")
	fs.BoolVar(&o.Yes, "yes", false, "accepted for scripts; this install never asks")
	fs.BoolVar(&o.Yes, "y", false, "shorthand for --yes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return h.InstallHost(ctx, o)
}

// InstallHost installs a runner or automations host on this machine: it enrolls with the instance for its own
// credential, then runs as its own service from its own directory.
func (h *Host) InstallHost(ctx context.Context, o HostOptions) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	if err := o.validate(); err != nil {
		return err
	}
	tag, err := installTag(o.Version)
	if err != nil {
		return err
	}
	h.printf("Nexul %s %s installer\n\n", tag, o.Kind)
	if o.Kind == kindRunner {
		for _, s := range h.prerequisites(ctx) {
			if err := h.step(s.label, s.run); err != nil {
				return err
			}
		}
	}
	if err := h.step("Command", h.installSelf); err != nil {
		return err
	}
	if o.Kind == kindAutomations && h.GOOS == "linux" {
		if err := h.step("User", func() (string, error) { return h.ensureServiceUser(ctx) }); err != nil {
			return err
		}
	}
	label := strings.ToUpper(o.Kind[:1]) + o.Kind[1:]
	if err := h.step(label, func() (string, error) { return h.installHost(ctx, o, tag) }); err != nil {
		return err
	}
	h.printf("\nThe %s %s is running as the %s service.\n", hostNoun(o.Kind), o.Name, unitName(o.Kind, o.Name))
	h.printf("  Remove it      nexul uninstall %s %s\n  Status         nexul status\n", o.Kind, o.Name)
	return nil
}

func (o HostOptions) validate() error {
	if !hostNamePattern.MatchString(o.Name) {
		return fmt.Errorf("--name %q is not a valid name: use 1 to 32 lower case letters, digits and dashes, starting with a letter or digit", o.Name)
	}
	if !strings.HasPrefix(o.Server, "http://") && !strings.HasPrefix(o.Server, "https://") {
		return fmt.Errorf("--server %q must be the instance's http:// or https:// address", o.Server)
	}
	if o.Code == "" {
		return errors.New("--code is required: copy the whole command from the instance's Add " + o.Kind + " dialog")
	}
	return nil
}

// installHost downloads the host's binary into its directory, trades the enrollment code for its credential and
// starts its service. A failure removes the directory again, so the next attempt starts clean.
func (h *Host) installHost(ctx context.Context, o HostOptions, tag string) (string, error) {
	u := h.newUnit(o.Kind, o.Name, tag)
	existing, err := h.loadUnit(u.Name)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return "", fmt.Errorf("a %s named %s is already installed here; remove it first with: nexul uninstall %s %s", o.Kind, o.Name, o.Kind, o.Name)
	}
	if err := h.setUpHost(ctx, &u, o, tag); err != nil {
		return "", errors.Join(err, os.RemoveAll(u.Dir))
	}
	return u.Name, nil
}

func (h *Host) setUpHost(ctx context.Context, u *Unit, o HostOptions, tag string) error {
	if err := os.MkdirAll(u.Dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", u.Dir, err)
	}
	if err := h.releaseAsset(ctx, tag, h.assetName(binaryFor(o.Kind)), u.Exec); err != nil {
		return err
	}
	server := strings.TrimSuffix(o.Server, "/")
	if o.Kind == kindRunner && o.StackRoot == "" {
		o.StackRoot = u.Dir
	}
	credential, err := h.enroll(ctx, server, o, tag)
	if err != nil {
		return err
	}
	credFile := filepath.Join(u.Dir, "credential")
	if err := os.WriteFile(credFile, []byte(credential+"\n"), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", credFile, err)
	}
	u.Env = hostEnv(o, server, credFile, h.ctlPath())
	if o.Kind == kindAutomations && h.GOOS == "linux" {
		u.User = serviceUser
	}
	if err := h.saveUnit(*u); err != nil {
		return err
	}
	if u.User != "" {
		if err := h.giveToServiceUser(ctx, u.Dir); err != nil {
			return err
		}
		if err := h.writeSudoers(ctx, o.Name); err != nil {
			return err
		}
	}
	return h.services().install(ctx, *u)
}

// hostEnv is the env contract a runner or automations host reads.
func hostEnv(o HostOptions, server, credFile, ctl string) map[string]string {
	env := map[string]string{
		"NEXUL_SERVER_URL":      server,
		"NEXUL_CREDENTIAL_FILE": credFile,
		"NEXUL_CTL":             ctl,
	}
	if o.Kind == kindAutomations {
		env["NEXUL_AUTOMATIONS_HOST_NAME"] = o.Name
		return env
	}
	env["NEXUL_RUNNER_NAME"] = o.Name
	env["NEXUL_STACK_ROOT"] = o.StackRoot
	if o.GitToken != "" {
		env["NEXUL_GIT_TOKEN"] = o.GitToken
	}
	return env
}

// apiPath is the kind's collection under /api.
func apiPath(kind string) string {
	if kind == kindAutomations {
		return "/api/automation-hosts"
	}
	return "/api/runners"
}

type enrollResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Credential string `json:"credential"`
}

// enroll trades the one-time code for the host's own credential.
func (h *Host) enroll(ctx context.Context, server string, o HostOptions, tag string) (string, error) {
	body := map[string]string{"code": o.Code, "name": o.Name, "os": h.GOOS, "arch": h.GOARCH, "version": tag}
	if o.Kind == kindRunner {
		body["stack_root"] = o.StackRoot
	}
	if hostname, err := h.Hostname(); err == nil {
		body["machine"] = hostname
	}
	resp, err := h.postJSON(ctx, server+apiPath(o.Kind)+"/enroll", "", body)
	if err != nil {
		return "", fmt.Errorf("enroll with %s: %w", server, err)
	}
	defer func() { _ = resp.Body.Close() }() // read-only response body
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("enroll with %s: %w", server, err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("the instance refused the enrollment code: it is unknown, already used or expired; make a new one with Add %s", o.Kind)
	}
	if resp.StatusCode == http.StatusConflict {
		return "", fmt.Errorf("the enrollment code was made for a different name than %q; use the command exactly as the instance showed it", o.Name)
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("enroll with %s: the instance answered %s: %s", server, resp.Status, strings.TrimSpace(string(data)))
	}
	var got enrollResponse
	if err := json.Unmarshal(data, &got); err != nil || got.Credential == "" {
		return "", fmt.Errorf("enroll with %s: the instance's answer holds no credential", server)
	}
	return got.Credential, nil
}

func (h *Host) postJSON(ctx context.Context, url, bearer string, body any) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return h.HTTP.Do(req)
}

func (h *Host) runUninstallHost(ctx context.Context, kind string, args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: nexul uninstall %s <name> [--detach] [--yes]", kind)
	}
	name := args[0]
	fs := flag.NewFlagSet("uninstall "+kind, flag.ContinueOnError)
	fs.SetOutput(h.Out)
	detach := fs.Bool("detach", false, "run the removal outside the calling service and return at once")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if *detach {
		return h.detach(ctx, append([]string{"uninstall", kind}, args...))
	}
	return h.UninstallHost(ctx, kind, name, *yes)
}

// UninstallHost removes a runner or automations host from this machine and tells the instance it is gone.
func (h *Host) UninstallHost(ctx context.Context, kind, name string, yes bool) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	missing := fmt.Errorf("no %s named %q is installed on this machine; nexul status lists what is", kind, name)
	if !hostNamePattern.MatchString(name) {
		return missing
	}
	u, err := h.loadUnit(unitName(kind, name))
	if err != nil {
		return err
	}
	if u == nil {
		return missing
	}
	h.printf("This removes the %s %s from this machine and from the instance.\n", hostNoun(kind), name)
	if err := h.confirm(yes); err != nil {
		return err
	}
	h.printf("\n")
	if err := h.step(u.Name, func() (string, error) { return h.removeUnit(ctx, *u) }); err != nil {
		return err
	}
	h.printf("\nThe %s %s is uninstalled.\n", hostNoun(kind), name)
	return nil
}

// removeUnit takes a unit off the machine: its service, its sudoers drop-in and its directory.
func (h *Host) removeUnit(ctx context.Context, u Unit) (string, error) {
	detail := "removed"
	if u.Kind == kindRunner || u.Kind == kindAutomations {
		detail += h.tellInstance(ctx, u)
	}
	if err := h.services().remove(ctx, u); err != nil {
		return "", err
	}
	if u.Kind == kindAutomations && h.GOOS == "linux" {
		if err := removeIfPresent(h.sudoersPath(u.Host)); err != nil {
			return "", fmt.Errorf("remove the sudoers drop-in: %w", err)
		}
	}
	if err := os.RemoveAll(u.Dir); err != nil {
		return "", fmt.Errorf("remove %s: %w", u.Dir, err)
	}
	return detail, nil
}

// tellInstance asks the instance to drop the host, best-effort: a host whose instance is gone must still come off
// this machine.
func (h *Host) tellInstance(ctx context.Context, u Unit) string {
	credential, err := os.ReadFile(u.Env["NEXUL_CREDENTIAL_FILE"])
	if err != nil {
		return "; the instance was not told (no credential)"
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	url := u.Env["NEXUL_SERVER_URL"] + apiPath(u.Kind) + "/self/remove"
	resp, err := h.postJSON(ctx, url, strings.TrimSpace(string(credential)), struct{}{})
	if err != nil {
		return "; the instance was not told (" + err.Error() + ")"
	}
	_ = resp.Body.Close() // only the status matters
	if resp.StatusCode == http.StatusUnauthorized {
		return "; the instance had already removed it"
	}
	if resp.StatusCode/100 != 2 {
		return "; the instance was not told (it answered " + resp.Status + ")"
	}
	return "; the instance was told"
}

// ensureServiceUser creates the nexul system user the server, OpenObserve and automations hosts run as on Linux.
func (h *Host) ensureServiceUser(ctx context.Context) (string, error) {
	if _, err := h.Exec.Run(ctx, "id", "-u", serviceUser); err == nil {
		return serviceUser + ", present", nil
	}
	if _, err := h.Exec.Run(ctx, "useradd", "--system", "--user-group", "--no-create-home", "--shell", "/usr/sbin/nologin", serviceUser); err != nil {
		return "", fmt.Errorf("create the %s user: %w", serviceUser, err)
	}
	return serviceUser + ", created", nil
}

// removeServiceUser deletes the nexul system user and its group, the reverse of ensureServiceUser.
func (h *Host) removeServiceUser(ctx context.Context) (string, error) {
	if _, err := h.Exec.Run(ctx, "id", "-u", serviceUser); err != nil {
		return serviceUser + ", not present", nil
	}
	if _, err := h.Exec.Run(ctx, "userdel", serviceUser); err != nil {
		return "", fmt.Errorf("remove the %s user: %w", serviceUser, err)
	}
	return serviceUser + ", removed", nil
}

func (h *Host) giveToServiceUser(ctx context.Context, paths ...string) error {
	_, err := h.Exec.Run(ctx, "chown", append([]string{"-R", serviceUser + ":" + serviceUser}, paths...)...)
	return err
}

func (h *Host) sudoersPath(name string) string {
	return filepath.Join(h.Paths.Sudoers, "nexul-automations-"+name)
}

// writeSudoers lets the automations host's user run exactly its own removal as root, which is what its service
// does when the instance removes it.
func (h *Host) writeSudoers(ctx context.Context, name string) error {
	path := h.sudoersPath(name)
	rule := fmt.Sprintf("%s ALL=(root) NOPASSWD: %s uninstall automations %s --detach\n", serviceUser, h.ctlPath(), name)
	// sudo skips drop-in names containing a dot, so the file is inert until visudo has passed it.
	tmp := path + ".new"
	if err := os.WriteFile(tmp, []byte(rule), 0o440); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if _, err := h.Exec.Run(ctx, "visudo", "-cf", tmp); err != nil {
		return errors.Join(fmt.Errorf("the sudoers rule did not validate: %w", err), os.Remove(tmp))
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("install %s: %w", path, err)
	}
	return nil
}

// hostNoun is how the terminal names a kind of host: "automations" alone reads as the automations themselves.
func hostNoun(kind string) string {
	if kind == kindAutomations {
		return "automations host"
	}
	return kind
}
