package t3clientv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/t3rpc"
)

// guestTokenEnv is the instance environment variable a guest's Nexul token rides in, never a launch argument.
const guestTokenEnv = "NEXUL_GUEST_TOKEN"

// guestReadyWait bounds how long OpenGuest waits for T3 to list the instance it made.
const guestReadyWait = 30 * time.Second

// guestInstancePrefix starts every instance OpenGuest makes, so a leftover is recognisable as Nexul's.
const guestInstancePrefix = "nexul-guest-"

type providerInstance struct {
	Driver      string         `json:"driver"`
	DisplayName string         `json:"displayName,omitempty"`
	AccentColor string         `json:"accentColor,omitempty"`
	Environment []envVar       `json:"environment,omitempty"`
	Enabled     *bool          `json:"enabled,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
}

type envVar struct {
	Name          string `json:"name"`
	Value         string `json:"value"`
	Sensitive     bool   `json:"sensitive"`
	ValueRedacted bool   `json:"valueRedacted,omitempty"`
}

type instanceMutation struct {
	Operation  string            `json:"operation"`
	InstanceID string            `json:"instanceId"`
	Instance   *providerInstance `json:"instance,omitempty"`
}

type updateSettings struct {
	Patch    struct{}         `json:"patch"`
	Mutation instanceMutation `json:"providerInstanceMutation"`
}

// OpenGuest implements harness.GuestInstances: a copy of base's instance whose Nexul MCP server is reached with token.
func (h *Harness) OpenGuest(ctx context.Context, s harness.Session, base, mcpURL, token, label string) (id string, err error) {
	ok, err := t3rpc.CanManageProviders(ctx, h.Options.HTTPClient, s.ServerURL, s.BearerToken)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%w: Nexul's session with T3 Code on this computer predates running someone else's agents; pair the computer again", apperrs.ErrConflict)
	}
	c, err := h.connect(ctx, s)
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, c.Close())
	}()
	instances, err := readInstances(ctx, c)
	if err != nil {
		return "", err
	}
	inst, err := guestInstance(instances, base, mcpURL, token, label)
	if err != nil {
		return "", err
	}
	id = guestInstancePrefix + strings.ToLower(ids.New())
	if _, err := c.Call(ctx, "server.updateSettings", updateSettings{Mutation: instanceMutation{Operation: "create", InstanceID: id, Instance: &inst}}); err != nil {
		return "", fmt.Errorf("add a t3 provider instance: %w", err)
	}
	if err := h.awaitInstance(ctx, s, id); err != nil {
		return "", errors.Join(err, h.removeInstance(ctx, c, id))
	}
	return id, nil
}

// CloseGuest implements harness.GuestInstances.
func (h *Harness) CloseGuest(ctx context.Context, s harness.Session, instanceID string) (err error) {
	if !strings.HasPrefix(instanceID, guestInstancePrefix) {
		return fmt.Errorf("%w: %q is not an instance Nexul made", apperrs.ErrInvalid, instanceID)
	}
	c, err := h.connect(ctx, s)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, c.Close())
	}()
	return h.removeInstance(ctx, c, instanceID)
}

type instanceSettings struct {
	ProviderInstances map[string]providerInstance `json:"providerInstances"`
}

// readInstances reads the provider instances settings hold: from server.getConfig's settings where T3 sends them
// there, else from server.getSettings.
func readInstances(ctx context.Context, c *t3rpc.Conn) (map[string]providerInstance, error) {
	raw, err := c.Call(ctx, "server.getConfig", struct{}{})
	if err != nil {
		return nil, fmt.Errorf("read t3 config: %w", err)
	}
	var config struct {
		Settings *instanceSettings `json:"settings"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("decode t3 config: %w", err)
	}
	if config.Settings != nil {
		return config.Settings.ProviderInstances, nil
	}
	if raw, err = c.Call(ctx, "server.getSettings", struct{}{}); err != nil {
		return nil, fmt.Errorf("read t3 settings: %w", err)
	}
	var settings instanceSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("decode t3 settings: %w", err)
	}
	return settings.ProviderInstances, nil
}

func (h *Harness) removeInstance(ctx context.Context, c *t3rpc.Conn, id string) error {
	if _, err := c.Call(ctx, "server.updateSettings", updateSettings{Mutation: instanceMutation{Operation: "remove", InstanceID: id}}); err != nil {
		return fmt.Errorf("remove t3 provider instance %s: %w", id, err)
	}
	return nil
}

// awaitInstance waits for T3 to list the instance, which it builds after the settings change.
func (h *Harness) awaitInstance(ctx context.Context, s harness.Session, id string) error {
	ctx, cancel := context.WithTimeout(ctx, guestReadyWait)
	defer cancel()
	for {
		providers, err := h.ListProviders(ctx, s)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(providers, func(p harness.Provider) bool { return p.ID == id }) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: T3 Code did not start the provider for this run within %s", apperrs.ErrRetryable, guestReadyWait)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// guestInstance copies base's instance, or the built-in one T3 makes for a driver with no settings of its own, and
// points its Nexul MCP server at mcpURL with the token from the instance's own environment.
func guestInstance(instances map[string]providerInstance, base, mcpURL, token, label string) (providerInstance, error) {
	inst, ok := instances[base]
	if !ok {
		inst = providerInstance{Driver: base}
	}
	args, err := guestArgs(inst.Driver, mcpURL)
	if err != nil {
		return providerInstance{}, err
	}
	env := make([]envVar, 0, len(inst.Environment)+1)
	for _, v := range inst.Environment {
		if v.Sensitive || v.ValueRedacted {
			return providerInstance{}, fmt.Errorf("%w: %s on this computer keeps a secret setting Nexul cannot copy, so it cannot run someone else's agent", apperrs.ErrConflict, base)
		}
		if v.Name != guestTokenEnv {
			env = append(env, v)
		}
	}
	env = append(env, envVar{Name: guestTokenEnv, Value: token, Sensitive: true})
	config := maps.Clone(inst.Config)
	if config == nil {
		config = map[string]any{}
	}
	own, _ := config["launchArgs"].(string)
	config["launchArgs"] = strings.TrimSpace(own + " " + args)
	enabled := true
	return providerInstance{Driver: inst.Driver, DisplayName: label, AccentColor: inst.AccentColor, Environment: env, Enabled: &enabled, Config: config}, nil
}

// guestArgs are the launch arguments that give the driver's agent only the Nexul server reached with the guest's token.
func guestArgs(driver, mcpURL string) (string, error) {
	if strings.ContainsAny(mcpURL, `'" `) {
		return "", fmt.Errorf("%w: the MCP address %q holds a quote or a space", apperrs.ErrInvalid, mcpURL)
	}
	if driver == "claudeAgent" {
		server, err := json.Marshal(map[string]any{"mcpServers": map[string]any{"nexul": map[string]any{
			"type": "http", "url": mcpURL, "headers": map[string]string{"Authorization": "Bearer ${" + guestTokenEnv + "}"},
		}}})
		if err != nil {
			return "", fmt.Errorf("encode the guest MCP config: %w", err)
		}
		return "--strict-mcp-config --mcp-config '" + string(server) + "'", nil
	}
	if driver == "codex" {
		return `-c 'mcp_servers.nexul.url="` + mcpURL + `"' -c 'mcp_servers.nexul.http_headers={}' -c 'mcp_servers.nexul.bearer_token_env_var="` + guestTokenEnv + `"'`, nil
	}
	return "", fmt.Errorf("%w: running someone else's agent on a shared computer takes Claude or Codex, not %s", apperrs.ErrConflict, driver)
}
