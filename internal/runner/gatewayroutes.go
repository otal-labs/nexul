package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Where a proxy gateway's Traefik reads its file provider config from, and the env var holding the base the container
// was started with (the instance's own route, or nothing).
const (
	gatewayConfigEnv  = "NEXUL_TRAEFIK_DYNAMIC"
	gatewayConfigFile = "/etc/traefik/nexul.yml"
	setRoutesTimeout  = 20 * time.Second
)

// SetRoutes implements Executor.SetRoutes: rewrites the gateway's config file as its base plus routes, which Traefik's
// file provider reloads on change, and reports the outcome on a gateway_routes_result frame.
func (e *ShellExecutor) SetRoutes(ctx context.Context, gatewayContainer, routes string, send func(Frame)) {
	result := Frame{Type: FrameGatewayRoutesResult, GatewayContainer: gatewayContainer, Status: BuildStatusSuccess}
	if err := setGatewayRoutes(ctx, e.cmd, gatewayContainer, routes); err != nil {
		e.log.Warn("gateway_routes failed", "gateway_container", gatewayContainer, "error", err)
		result.Status = BuildStatusFailed
		result.Error = err.Error()
	}
	send(result)
}

// setGatewayRoutes reads the container's base config from its environment, merges routes into it and swaps the file in
// by rename, so Traefik never reads a half-written one.
func setGatewayRoutes(ctx context.Context, cmd CommandRunner, container, routes string) error {
	if cmd == nil {
		cmd = ShellCommandRunner
	}
	var base strings.Builder
	if err := cmd(ctx, "", func(l string) { base.WriteString(l) }, "docker", "exec", container, "printenv", gatewayConfigEnv); err != nil {
		return fmt.Errorf("read the gateway's base config: %w", err)
	}
	merged, err := mergeRoutes(base.String(), routes)
	if err != nil {
		return err
	}
	// ponytail: the config travels as one argument, capped near 128KiB by the kernel (about 300 exposures per gateway).
	script := `printf '%s' "$1" > ` + gatewayConfigFile + `.tmp && mv ` + gatewayConfigFile + `.tmp ` + gatewayConfigFile
	if err := cmd(ctx, "", discardLog, "docker", "exec", container, "sh", "-c", script, "sh", merged); err != nil {
		return fmt.Errorf("write the gateway's config: %w", err)
	}
	return nil
}

// mergeRoutes adds the routers and services of routes (an http fragment) to the base config; the base's own entries stay,
// and a section left empty is dropped.
func mergeRoutes(base, routes string) (string, error) {
	cfg := map[string]any{}
	if b := strings.TrimSpace(base); b != "" {
		if err := json.Unmarshal([]byte(b), &cfg); err != nil {
			return "", fmt.Errorf("parse the gateway's base config: %w", err)
		}
	}
	var frag map[string]map[string]any
	if err := json.Unmarshal([]byte(routes), &frag); err != nil {
		return "", fmt.Errorf("parse the gateway routes: %w", err)
	}
	httpCfg := objectAt(cfg, "http")
	for _, section := range []string{"routers", "services"} {
		target := objectAt(httpCfg, section)
		for name, v := range frag[section] {
			target[name] = v
		}
		// Traefik refuses a config holding an empty routers or services map and keeps serving the previous one.
		if len(target) == 0 {
			delete(httpCfg, section)
		}
	}
	if len(httpCfg) == 0 {
		delete(cfg, "http")
	}
	out, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("encode the gateway config: %w", err)
	}
	return string(out), nil
}

// objectAt returns m[key] as an object, adding an empty one when it is missing.
func objectAt(m map[string]any, key string) map[string]any {
	if o, ok := m[key].(map[string]any); ok {
		return o
	}
	o := map[string]any{}
	m[key] = o
	return o
}
