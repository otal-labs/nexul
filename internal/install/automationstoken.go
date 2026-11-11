package install

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// workspaceTokenPrefix starts the env variable of an automations host's token for one workspace's 1Password; the
// automations host reads the same prefix (automations/src/workspace-tokens.ts).
const workspaceTokenPrefix = "NEXUL_OP_TOKEN_"

const automationsUsage = "usage: nexul automations 1password <host> --workspace <slug> | --remove   (the token is read from standard input)"

// workspaceSlugPattern is a workspace slug: lower case words joined by single dashes.
var workspaceSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// workspaceTokenEnv is the variable holding the token for the workspace with id: the id's bytes in upper case hex, so
// any id makes a valid variable name that maps back to that one id. It is the id and not the slug, which a rename can
// hand to another workspace.
func workspaceTokenEnv(workspaceID string) string {
	return workspaceTokenPrefix + strings.ToUpper(hex.EncodeToString([]byte(workspaceID)))
}

// tokenWorkspace is the workspace id the variable key holds a token for, or false for a name workspaceTokenEnv never makes.
func tokenWorkspace(key string) (string, bool) {
	encoded, ok := strings.CutPrefix(key, workspaceTokenPrefix)
	if !ok || encoded == "" || encoded != strings.ToUpper(encoded) {
		return "", false
	}
	id, err := hex.DecodeString(encoded)
	if err != nil {
		return "", false
	}
	return string(id), true
}

// runAutomations is `nexul automations <manager> <host> --workspace <slug>`: one subcommand per secrets manager.
func (h *Host) runAutomations(ctx context.Context, args []string) error {
	if len(args) < 2 || args[0] != "1password" || strings.HasPrefix(args[1], "-") {
		return errors.New(automationsUsage)
	}
	name := args[1]
	fs := flag.NewFlagSet("automations 1password", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	remove := fs.Bool("remove", false, "remove the host's 1Password token instead of setting one")
	workspace := fs.String("workspace", "", "the slug of the workspace the token reads secrets for")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("unexpected argument: the token is read from standard input, never taken from the command line")
	}
	if *remove && *workspace != "" {
		return errors.New("--remove takes no --workspace: a host holds one workspace's token, and --remove removes it")
	}
	if *remove {
		return h.RemoveAutomationsToken(ctx, name)
	}
	if !workspaceSlugPattern.MatchString(*workspace) {
		return errors.New("--workspace is required and takes a workspace slug: lower case letters, digits and dashes")
	}
	if err := h.checkUser(); err != nil {
		return err
	}
	if _, err := h.hostUnit(kindAutomations, name); err != nil {
		return err
	}
	token, err := h.readStdinToken()
	if err != nil {
		return err
	}
	return h.SetAutomationsToken(ctx, name, *workspace, token)
}

// SetAutomationsToken stores token as the named automations host's 1Password token for the workspace with slug, keyed
// by the workspace's id, and restarts the host to pick it up. Every automation on a host can read its tokens, so a host
// holds one workspace's token and runs only that workspace's automations (ADR 0182): a token for a second workspace,
// or for a host running another workspace's automations, is refused.
func (h *Host) SetAutomationsToken(ctx context.Context, name, slug, token string) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	u, err := h.hostUnit(kindAutomations, name)
	if err != nil {
		return err
	}
	got, err := h.hostWorkspace(ctx, u, slug)
	if err != nil {
		return err
	}
	for key := range u.Env {
		held, ok := tokenWorkspace(key)
		if ok && held != got.Workspace.ID {
			return fmt.Errorf("the automations host %s already holds a 1Password token for another workspace (%s); a host holds one workspace's token, so remove that one first with --remove", name, held)
		}
	}
	for _, p := range got.Placed {
		if p.ID != got.Workspace.ID {
			return fmt.Errorf("the automations host %s has automations of the workspace %s placed on it; a host with a 1Password token runs one workspace's automations only, so move those to another host first", name, p.Slug)
		}
	}
	return h.setUnitToken(ctx, u, workspaceTokenEnv(got.Workspace.ID), fmt.Sprintf("the automations host %s for the workspace %s", name, slug), token)
}

// RemoveAutomationsToken removes the named automations host's 1Password token, whichever workspace it is for, and
// restarts the host. It needs no instance, so a token comes off a host whose instance is unreachable.
func (h *Host) RemoveAutomationsToken(ctx context.Context, name string) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	u, err := h.hostUnit(kindAutomations, name)
	if err != nil {
		return err
	}
	var keys []string
	for key := range u.Env {
		if strings.HasPrefix(key, workspaceTokenPrefix) {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		h.printf("The automations host %s has no 1Password token.\n", name)
		return nil
	}
	for _, key := range keys[1:] {
		delete(u.Env, key)
	}
	return h.setUnitToken(ctx, u, keys[0], "the automations host "+name, "")
}

// hostWorkspaceLookup is the instance's GET /api/automation-hosts/self/workspace answer.
type hostWorkspaceLookup struct {
	Workspace hostWorkspace   `json:"workspace"`
	Placed    []hostWorkspace `json:"placed"`
}

type hostWorkspace struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

// hostWorkspace asks the instance, as the host with its own credential, for the id of the workspace slug names now and
// the workspaces whose automations are placed on the host.
func (h *Host) hostWorkspace(ctx context.Context, u *Unit, slug string) (hostWorkspaceLookup, error) {
	credential, err := os.ReadFile(u.Env["NEXUL_CREDENTIAL_FILE"])
	if err != nil {
		return hostWorkspaceLookup{}, fmt.Errorf("read the host's credential: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	endpoint := u.Env["NEXUL_SERVER_URL"] + apiPath(kindAutomations) + "/self/workspace?slug=" + url.QueryEscape(slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return hostWorkspaceLookup{}, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(credential)))
	resp, err := h.HTTP.Do(req)
	if err != nil {
		return hostWorkspaceLookup{}, fmt.Errorf("ask the instance which workspace %s is: %w", slug, err)
	}
	defer func() { _ = resp.Body.Close() }() // read-only response body
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return hostWorkspaceLookup{}, fmt.Errorf("ask the instance which workspace %s is: %w", slug, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return hostWorkspaceLookup{}, fmt.Errorf("the instance has no workspace with the slug %q", slug)
	}
	if resp.StatusCode != http.StatusOK {
		return hostWorkspaceLookup{}, fmt.Errorf("ask the instance which workspace %s is: it answered %s", slug, resp.Status)
	}
	var got hostWorkspaceLookup
	if err := json.Unmarshal(data, &got); err != nil || got.Workspace.ID == "" {
		return hostWorkspaceLookup{}, fmt.Errorf("ask the instance which workspace %s is: its answer names no workspace", slug)
	}
	return got, nil
}

// workspaceTokenNote is the status column for an automations host: which workspace holds a token, never a token.
func (u Unit) workspaceTokenNote() string {
	var ids []string
	for key := range u.Env {
		if id, ok := tokenWorkspace(key); ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "no 1Password token"
	}
	sort.Strings(ids)
	return "1Password token for workspace " + strings.Join(ids, ", ")
}
