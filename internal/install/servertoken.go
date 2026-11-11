package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
)

const serverUsage = "usage: nexul server 1password [--remove]   (the token is read from standard input)"

var errNoServer = errors.New("no Nexul server is installed on this machine; run nexul install")

// runServer is `nexul server <manager>`: one subcommand per secrets manager, like `nexul runner`.
func (h *Host) runServer(ctx context.Context, args []string) error {
	if len(args) < 1 || args[0] != "1password" {
		return errors.New(serverUsage)
	}
	fs := flag.NewFlagSet("server 1password", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	remove := fs.Bool("remove", false, "remove the server's 1Password token instead of setting one")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("unexpected argument: the token is read from standard input, never taken from the command line")
	}
	if *remove {
		return h.SetServerToken(ctx, "")
	}
	if err := h.checkUser(); err != nil {
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
	return h.SetServerToken(ctx, token)
}

// SetServerToken stores token as the server's 1Password token, or removes it when token is empty, and restarts the
// server to pick it up. It lands in the install's 0600 .env, which every upgrade rewrites and keeps, and in the unit's
// env the server reads; it never goes into a command's arguments.
func (h *Host) SetServerToken(ctx context.Context, token string) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	prev, err := h.loadInstall()
	if err != nil {
		return err
	}
	u, err := h.loadUnit(unitName(kindServer, ""))
	if err != nil {
		return err
	}
	if prev == nil || u == nil {
		return errNoServer
	}
	if token == "" && !holdsToken(prev.Env) && !holdsToken(u.Env) {
		h.printf("The server has no 1Password token.\n")
		return nil
	}
	prev.Env = withToken(prev.Env, token)
	u.Env = withToken(u.Env, token)
	if err := writeEnvFile(filepath.Join(prev.Dir, ".env"), prev.Env); err != nil {
		return err
	}
	if err := h.saveUnit(*u); err != nil {
		return err
	}
	if err := h.services().install(ctx, *u); err != nil {
		return err
	}
	if token == "" {
		h.printf("Removed the 1Password token from the server and restarted it.\n")
		return nil
	}
	h.printf("Stored the 1Password token for the server and restarted it.\n")
	return nil
}

func holdsToken(env map[string]string) bool {
	_, ok := env[opTokenEnv]
	return ok
}

// withToken is env with the token set, or without it when token is empty.
func withToken(env map[string]string, token string) map[string]string {
	if env == nil {
		env = map[string]string{}
	}
	if token == "" {
		delete(env, opTokenEnv)
		return env
	}
	env[opTokenEnv] = token
	return env
}
