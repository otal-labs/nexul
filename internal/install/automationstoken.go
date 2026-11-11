package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// workspaceTokenPrefix starts the env variable of an automations host's token for one workspace's 1Password; the
// automations host reads the same prefix (automations/src/workspace-tokens.ts).
const workspaceTokenPrefix = "NEXUL_OP_TOKEN_"

const automationsUsage = "usage: nexul automations 1password <host> --workspace <slug> [--remove]   (the token is read from standard input)"

// workspaceSlugPattern is a workspace slug: lower case words joined by single dashes.
var workspaceSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// workspaceTokenEnv is the variable holding the token for the workspace with slug. A slug holds only lower case
// letters, digits and dashes, so upper case with underscores for dashes is a valid name that maps back to one slug.
func workspaceTokenEnv(slug string) string {
	return workspaceTokenPrefix + strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
}

// runAutomations is `nexul automations <manager> <host> --workspace <slug>`: one subcommand per secrets manager.
func (h *Host) runAutomations(ctx context.Context, args []string) error {
	if len(args) < 2 || args[0] != "1password" || strings.HasPrefix(args[1], "-") {
		return errors.New(automationsUsage)
	}
	name := args[1]
	fs := flag.NewFlagSet("automations 1password", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	remove := fs.Bool("remove", false, "remove the host's 1Password token for the workspace instead of setting one")
	workspace := fs.String("workspace", "", "the slug of the workspace the token reads secrets for")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("unexpected argument: the token is read from standard input, never taken from the command line")
	}
	if !workspaceSlugPattern.MatchString(*workspace) {
		return errors.New("--workspace is required and takes a workspace slug: lower case letters, digits and dashes")
	}
	if *remove {
		return h.SetAutomationsToken(ctx, name, *workspace, "")
	}
	if err := h.checkUser(); err != nil {
		return err
	}
	if _, err := h.hostUnit(kindAutomations, name); err != nil {
		return err
	}
	if _, err := h.readStdinToken(); err != nil {
		return err
	}
	// shortcut: storing is refused until a host holds tokens for one workspace only (ADR 0182), then this goes.
	return errors.New("an automations host can't hold a 1Password token yet, because automations on a shared host could read it; --remove still removes one already set")
}

// SetAutomationsToken stores token as the named automations host's 1Password token for a workspace, or removes it
// when token is empty, and restarts the host to pick it up.
func (h *Host) SetAutomationsToken(ctx context.Context, name, workspace, token string) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	u, err := h.hostUnit(kindAutomations, name)
	if err != nil {
		return err
	}
	return h.setUnitToken(ctx, u, workspaceTokenEnv(workspace), fmt.Sprintf("the automations host %s for the workspace %s", name, workspace), token)
}

// workspaceTokenNote is the status column for an automations host: which workspaces hold a token, never a token.
func (u Unit) workspaceTokenNote() string {
	var slugs []string
	for key := range u.Env {
		name, ok := strings.CutPrefix(key, workspaceTokenPrefix)
		if !ok {
			continue
		}
		slugs = append(slugs, strings.ToLower(strings.ReplaceAll(name, "_", "-")))
	}
	if len(slugs) == 0 {
		return "no 1Password token"
	}
	sort.Strings(slugs)
	return "1Password token for " + strings.Join(slices.Compact(slugs), ", ")
}
