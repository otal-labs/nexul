package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// opTokenEnv is the variable a 1Password service account token travels in, the one name the 1Password SDK reads.
const opTokenEnv = "OP_SERVICE_ACCOUNT_TOKEN"

const runnerUsage = "usage: nexul runner 1password <name> [--remove]   (the token is read from standard input)"

// runRunner is `nexul runner <manager> <name>`: one subcommand per secrets manager, since each manager's credential
// has its own shape.
func (h *Host) runRunner(ctx context.Context, args []string) error {
	if len(args) < 2 || args[0] != "1password" || strings.HasPrefix(args[1], "-") {
		return errors.New(runnerUsage)
	}
	name := args[1]
	fs := flag.NewFlagSet("runner 1password", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	remove := fs.Bool("remove", false, "remove the runner's 1Password token instead of setting one")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("unexpected argument: the token is read from standard input, never taken from the command line")
	}
	if *remove {
		return h.SetRunnerToken(ctx, name, "")
	}
	if err := h.checkUser(); err != nil {
		return err
	}
	if _, err := h.runnerUnit(name); err != nil {
		return err
	}
	if h.Interactive {
		h.printf("Paste the 1Password service account token, then press Enter.\n")
	}
	line, err := h.In.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read the token: %w", err)
	}
	token, err := cleanToken(line)
	if err != nil {
		return err
	}
	return h.SetRunnerToken(ctx, name, token)
}

// cleanToken trims a token read from stdin or a file and refuses one that could not sit on one env line.
func cleanToken(raw string) (string, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return "", errors.New("no token given: pipe the 1Password service account token in, or pass --remove")
	}
	if strings.ContainsFunc(token, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return "", errors.New("that is not a service account token: it spans more than one line or holds spaces")
	}
	return token, nil
}

// readTokenFile reads the token an install is given by file, so it never rides a command line.
func readTokenFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read --1password-token-file: %w", err)
	}
	token, err := cleanToken(string(data))
	if err != nil {
		return "", fmt.Errorf("--1password-token-file: %w", err)
	}
	return token, nil
}

// SetRunnerToken stores token as the named runner's 1Password token, or removes it when token is empty, and restarts
// the runner to pick it up. The token only ever goes into the unit's own env, never into a command's arguments.
func (h *Host) SetRunnerToken(ctx context.Context, name, token string) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	u, err := h.runnerUnit(name)
	if err != nil {
		return err
	}
	_, had := u.Env[opTokenEnv]
	if token == "" && !had {
		h.printf("The runner %s has no 1Password token.\n", name)
		return nil
	}
	if u.Env == nil {
		u.Env = map[string]string{}
	}
	if token == "" {
		delete(u.Env, opTokenEnv)
	}
	if token != "" {
		u.Env[opTokenEnv] = token
	}
	if err := h.saveUnit(*u); err != nil {
		return err
	}
	if err := h.services().install(ctx, *u); err != nil {
		return err
	}
	if token == "" {
		h.printf("Removed the 1Password token from the runner %s and restarted it.\n", name)
		return nil
	}
	h.printf("Stored the 1Password token for the runner %s and restarted it.\n", name)
	return nil
}

// runnerUnit is the installed runner unit called name, or the error that says there is none.
func (h *Host) runnerUnit(name string) (*Unit, error) {
	missing := fmt.Errorf("no runner named %q is installed on this machine; nexul status lists what is", name)
	if !hostNamePattern.MatchString(name) {
		return nil, missing
	}
	u, err := h.loadUnit(unitName(kindRunner, name))
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, missing
	}
	return u, nil
}

// tokenNote is the status column that says whether a runner holds a 1Password token, never the token.
func (u Unit) tokenNote() string {
	if u.Kind != kindRunner {
		return ""
	}
	if _, ok := u.Env[opTokenEnv]; ok {
		return "1Password token set"
	}
	return "no 1Password token"
}
