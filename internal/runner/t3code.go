package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/t3rpc"
)

// T3 Code's states as a personal runner finds them on its computer.
const (
	T3Answering   = "answering"
	T3NotRunning  = "not_running"
	T3Missing     = "missing"
	T3NotLoopback = "not_loopback"
)

const (
	// factsInterval paces a personal runner's facts reports after the one it sends on connect.
	factsInterval = 6 * time.Hour
	// pairTokenTimeout bounds minting a pairing token: T3 Code's command starts a Node process on the computer.
	pairTokenTimeout = 30 * time.Second
)

// Facts is what a personal runner reports about its computer.
type Facts struct {
	Hostname string  `json:"hostname,omitempty" jsonschema:"The computer's hostname."`
	T3       T3Facts `json:"t3"`
}

// T3Facts is T3 Code on the computer as its runner found it.
type T3Facts struct {
	State   string `json:"state" enum:"answering,not_running,missing,not_loopback" jsonschema:"answering: T3 Code answers on loopback; not_running: installed but nothing answers; missing: not installed; not_loopback: bound to an address the runner refuses to reach."`
	Port    int    `json:"port,omitempty" jsonschema:"The loopback port T3 Code answers on; sent while answering."`
	Version string `json:"version,omitempty" jsonschema:"T3 Code's server version; sent while answering."`
}

// ---- server side ----

// pairWaiter is PairingToken waiting on the runner it asked for a token.
type pairWaiter struct {
	runnerID string
	result   chan Frame
}

// PairingToken asks computerID's connected personal runner for a one-time T3 Code pairing token minted on the computer.
func (h *Handler) PairingToken(ctx context.Context, computerID string) (string, error) {
	c, err := h.computerConn(ctx, computerID)
	if err != nil {
		return "", err
	}
	id := ids.New()
	w := pairWaiter{runnerID: c.id, result: make(chan Frame, 1)}
	h.pairMu.Lock()
	h.pairWaiters[id] = w
	h.pairMu.Unlock()
	defer func() {
		h.pairMu.Lock()
		delete(h.pairWaiters, id)
		h.pairMu.Unlock()
	}()
	if err := h.sendFrame(ctx, c, Frame{Type: FrameT3PairTokenRequest, ID: id}); err != nil {
		return "", apperrs.Retryable(fmt.Errorf("ask the runner of computer %s for a pairing token: %w", computerID, err))
	}
	timer := time.NewTimer(pairTokenTimeout)
	defer timer.Stop()
	select {
	case f := <-w.result:
		if f.Error != "" {
			return "", apperrs.Retryable(errors.New(f.Error))
		}
		return f.Token, nil
	case <-timer.C:
		return "", apperrs.Retryable(fmt.Errorf("the runner of computer %s sent no pairing token within %s", computerID, pairTokenTimeout))
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// deliverPairToken hands a t3_pair_token to the PairingToken waiting on it, only if this runner was the one asked.
func (h *Handler) deliverPairToken(c *runnerConn, f Frame) {
	h.pairMu.Lock()
	w, ok := h.pairWaiters[f.ID]
	if ok && w.runnerID == c.id {
		delete(h.pairWaiters, f.ID)
	}
	h.pairMu.Unlock()
	if !ok || w.runnerID != c.id {
		h.log.Warn("pairing token answer refused", "runner_id", c.id, "id", f.ID)
		return
	}
	w.result <- f
}

// publishFacts hands a personal runner's report to the computer's owner's domain; a deploy runner's is dropped.
func (h *Handler) publishFacts(ctx context.Context, c *runnerConn, f Frame) error {
	if !c.personal {
		h.log.Warn("facts from a runner that is not personal", "runner_id", c.id)
		return nil
	}
	return h.bus.Publish(ctx, TopicFactsReported, FactsReportedEvent{
		RunnerID: c.id, ComputerID: c.record.ComputerID, UserID: c.record.OwnerUserID, Facts: *f.Facts, MembersOnly: true,
	})
}

// ---- runner side ----

// factsLoop reports the computer's facts on connect and every factsInterval until the connection ends.
func (c *Client) factsLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(factsInterval)
	defer ticker.Stop()
	for {
		c.sendFrame(ctx, conn, Frame{Type: FrameFacts, Facts: c.facts(ctx)})
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Client) facts(ctx context.Context) *Facts {
	hostname, err := os.Hostname()
	if err != nil {
		c.log.Warn("read hostname", "error", err)
	}
	return &Facts{Hostname: hostname, T3: c.t3Facts(ctx)}
}

// t3Facts finds T3 Code: the port its runtime file names, whether it answers there, and its command.
func (c *Client) t3Facts(ctx context.Context) T3Facts {
	addr, err := t3Address(c.cfg.T3Home, defaultT3Home())
	if errors.Is(err, errNotLoopback) {
		return T3Facts{State: T3NotLoopback}
	}
	if err == nil {
		if version, err := c.probeT3(ctx, addr); err == nil {
			ap, _ := netip.ParseAddrPort(addr) // t3Address joined a loopback IP and a valid port
			return T3Facts{State: T3Answering, Port: int(ap.Port()), Version: version}
		}
	}
	if c.findT3(c.cfg.T3Home) == "" {
		return T3Facts{State: T3Missing}
	}
	return T3Facts{State: T3NotRunning}
}

// probeT3 reads T3 Code's version from its unauthenticated descriptor at addr, a loopback address, with no proxy.
func (c *Client) probeT3(ctx context.Context, addr string) (string, error) {
	client := &http.Client{Timeout: c.cfg.ConnectTimeout, Transport: &http.Transport{}}
	defer client.CloseIdleConnections()
	d, err := t3rpc.Describe(ctx, client, "http://"+addr)
	if err != nil {
		return "", err
	}
	return d.ServerVersion, nil
}

// answerPairToken mints a pairing token with T3 Code's own command and sends it back; the token is never logged.
func (c *Client) answerPairToken(ctx context.Context, conn *websocket.Conn, id string) {
	c.streams.Go(func() {
		defer c.recoverStream(id)
		token, err := c.mintPairToken(ctx)
		if err != nil {
			c.log.Warn("pairing token not minted", "runner", c.cfg.Name, "id", id, "error", err)
			c.sendFrame(ctx, conn, Frame{Type: FrameT3PairToken, ID: id, Error: err.Error()})
			return
		}
		c.sendFrame(ctx, conn, Frame{Type: FrameT3PairToken, ID: id, Token: token})
	})
}

func (c *Client) mintPairToken(ctx context.Context) (string, error) {
	if !c.cfg.Personal {
		return "", errors.New("this runner is not a personal runner")
	}
	t3 := c.findT3(c.cfg.T3Home)
	if t3 == "" {
		return "", errors.New("T3 Code's t3 command is not installed on this computer")
	}
	ctx, cancel := context.WithTimeout(ctx, pairTokenTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, t3, "auth", "pairing", "create", "--json", "--ttl", "5m", "--label", "Nexul")
	cmd.Env = append(os.Environ(), "T3CODE_HOME="+c.cfg.T3Home)
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return "", fmt.Errorf("t3 auth pairing create failed (%s): %s", exit.ProcessState, lastLine(exit.Stderr))
	}
	if err != nil {
		return "", fmt.Errorf("run t3 auth pairing create: %w", err)
	}
	var issued struct {
		Credential string `json:"credential"`
	}
	if err := json.Unmarshal(out, &issued); err != nil || issued.Credential == "" {
		return "", errors.New("t3 auth pairing create printed no token")
	}
	return issued.Credential, nil
}

// lastLine is the last non-empty line of a command's error output, which is where a CLI says what went wrong.
func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	line := strings.TrimSpace(lines[len(lines)-1])
	if len(line) > 300 {
		line = line[:300]
	}
	return line
}

// findT3 is T3 Code's command: t3 on PATH, else the desktop app's launcher in home, else the command-line install.
func findT3(home string) string {
	if path, err := exec.LookPath("t3"); err == nil {
		return path
	}
	candidates := []string{filepath.Join(home, "bin", "t3")}
	if userHome, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(userHome, ".local", "bin", "t3"))
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return path
		}
	}
	return ""
}
