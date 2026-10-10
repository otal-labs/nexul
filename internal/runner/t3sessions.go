package runner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"time"
)

// nexulSessionLabel is the client label Nexul pairs with, which names its sessions in T3 Code.
const nexulSessionLabel = "Nexul"

// revokeNexulSessions ends every T3 Code session Nexul paired, so a removed computer leaves Nexul no way back into
// T3 Code. Best effort: a computer without T3 Code, or a T3 Code that refuses, still uninstalls.
func (c *Client) revokeNexulSessions(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	t3 := c.findT3(c.cfg.T3Home)
	if t3 == "" {
		c.log.Info("T3 Code not found; no session to revoke")
		return
	}
	run := func(args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, t3, args...)
		cmd.Env = append(os.Environ(), "T3CODE_HOME="+c.cfg.T3Home)
		return cmd
	}
	out, err := run("auth", "session", "list", "--json").Output()
	if err != nil {
		c.log.Warn("listing T3 Code sessions failed", "error", err)
		return
	}
	var sessions []struct {
		SessionID string `json:"sessionId"`
		Client    struct {
			Label string `json:"label"`
		} `json:"client"`
	}
	if err := json.Unmarshal(out, &sessions); err != nil {
		c.log.Warn("reading T3 Code sessions failed", "error", err)
		return
	}
	for _, s := range sessions {
		if s.Client.Label != nexulSessionLabel || s.SessionID == "" {
			continue
		}
		if err := run("auth", "session", "revoke", s.SessionID).Run(); err != nil {
			c.log.Warn("revoking a T3 Code session failed", "session", s.SessionID, "error", err)
		}
	}
}
