package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	// t3ProbeInterval paces the probes that keep T3 Code's background service running.
	t3ProbeInterval = 30 * time.Second
	// t3MissesToRestart is how many probes in a row a background service misses before the runner restarts it.
	t3MissesToRestart = 2
	// t3RestartGap spaces restarts, so a T3 Code that cannot start is not restarted in a loop.
	t3RestartGap = 5 * time.Minute
	// t3RestartTimeout bounds `t3 service restart`, which stops the service and waits for it to start again.
	t3RestartTimeout = time.Minute
)

// Facts is what a personal runner reports about its computer.
type Facts struct {
	Hostname string  `json:"hostname,omitempty" jsonschema:"The computer's hostname."`
	T3       T3Facts `json:"t3"`
}

// T3Facts is T3 Code on the computer as its runner found it.
type T3Facts struct {
	State        string     `json:"state" enum:"answering,not_running,missing,not_loopback" jsonschema:"answering: T3 Code answers on loopback; not_running: installed but nothing answers; missing: not installed; not_loopback: bound to an address the runner refuses to reach."`
	Port         int        `json:"port,omitempty" jsonschema:"The loopback port T3 Code answers on; sent while answering."`
	Version      string     `json:"version,omitempty" jsonschema:"T3 Code's server version; sent while answering."`
	RestartedAt  *time.Time `json:"restarted_at,omitempty" jsonschema:"When the runner last restarted T3 Code's background service after it stopped answering."`
	RestartError string     `json:"restart_error,omitempty" jsonschema:"Why the runner could not restart T3 Code's background service, such as the person's user service manager not running; cleared once T3 Code answers."`
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

// factsLoop reports the computer's facts on connect, every factsInterval, and at once when T3 Code's state changes,
// until the connection ends.
func (c *Client) factsLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(factsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.t3.changed: // the report about to go out already carries the change
		default:
		}
		c.sendFrame(ctx, conn, Frame{Type: FrameFacts, Facts: c.facts(ctx)})
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-c.t3.changed:
		}
	}
}

func (c *Client) facts(ctx context.Context) *Facts {
	hostname, err := os.Hostname()
	if err != nil {
		c.log.Warn("read hostname", "error", err)
	}
	return &Facts{Hostname: hostname, T3: c.t3.withRestart(c.t3Probe(ctx).facts)}
}

// t3Probed is one look at T3 Code: what facts report, and whether its own background service runs it.
type t3Probed struct {
	facts          T3Facts
	serviceManaged bool
}

// t3Probe finds T3 Code: the port its runtime file names, whether it answers there, and its command.
func (c *Client) t3Probe(ctx context.Context) t3Probed {
	addr, serviceManaged, err := t3Runtime(c.cfg.T3Home, defaultT3Home())
	if errors.Is(err, errNotLoopback) {
		return t3Probed{facts: T3Facts{State: T3NotLoopback}}
	}
	if err == nil {
		if version, err := c.probeT3(ctx, addr); err == nil {
			ap, _ := netip.ParseAddrPort(addr) // t3Runtime joined a loopback IP and a valid port
			return t3Probed{facts: T3Facts{State: T3Answering, Port: int(ap.Port()), Version: version}, serviceManaged: serviceManaged}
		}
	}
	if c.findT3(c.cfg.T3Home) == "" {
		return t3Probed{facts: T3Facts{State: T3Missing}}
	}
	return t3Probed{facts: T3Facts{State: T3NotRunning}}
}

// t3Keeper probes T3 Code every t3ProbeInterval and restarts a background service that stopped answering. A T3 Code
// it never saw answering under its own service, such as the desktop app, is only reported.
type t3Keeper struct {
	probe   func(ctx context.Context) t3Probed
	restart func(ctx context.Context) error
	log     *slog.Logger
	// changed wakes the connection's facts loop to report a change at once.
	changed chan struct{}

	// Only run's goroutine touches these.
	misses         int
	serviceManaged bool
	attemptedAt    time.Time

	mu sync.Mutex
	// probed is false until the first probe, whose state the report on connect already carries.
	probed      bool
	last        T3Facts
	restartedAt time.Time
	restartErr  string
}

func newT3Keeper(probe func(context.Context) t3Probed, restart func(context.Context) error, log *slog.Logger) *t3Keeper {
	return &t3Keeper{probe: probe, restart: restart, log: log, changed: make(chan struct{}, 1)}
}

func (k *t3Keeper) run(ctx context.Context) {
	ticker := time.NewTicker(t3ProbeInterval)
	defer ticker.Stop()
	for {
		k.check(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// check probes once, restarts the service when it is due, and signals changed when what facts report moved.
func (k *t3Keeper) check(ctx context.Context) {
	p := k.probe(ctx)
	k.misses++
	if p.facts.State != T3NotRunning {
		k.misses = 0
	}
	if p.facts.State == T3Answering {
		k.serviceManaged = p.serviceManaged
		k.mu.Lock()
		k.restartErr = ""
		k.mu.Unlock()
	}
	if k.misses >= t3MissesToRestart && k.serviceManaged && (k.attemptedAt.IsZero() || time.Since(k.attemptedAt) >= t3RestartGap) {
		k.attemptedAt = time.Now()
		k.restartService(ctx)
	}
	k.mu.Lock()
	now := k.withRestartLocked(p.facts)
	moved := k.probed && !reflect.DeepEqual(now, k.last)
	k.last, k.probed = now, true
	k.mu.Unlock()
	if !moved {
		return
	}
	select {
	case k.changed <- struct{}{}:
	default:
	}
}

func (k *t3Keeper) restartService(ctx context.Context) {
	err := k.restart(ctx)
	k.mu.Lock()
	defer k.mu.Unlock()
	if err != nil {
		k.log.Warn("T3 Code's background service stopped answering and was not restarted", "error", err)
		k.restartErr = err.Error()
		return
	}
	k.log.Info("T3 Code's background service stopped answering; restarted it")
	k.restartedAt, k.restartErr = time.Now(), ""
}

// withRestart adds the last restart, or why it failed, to facts.
func (k *t3Keeper) withRestart(facts T3Facts) T3Facts {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.withRestartLocked(facts)
}

func (k *t3Keeper) withRestartLocked(facts T3Facts) T3Facts {
	if !k.restartedAt.IsZero() {
		at := k.restartedAt
		facts.RestartedAt = &at
	}
	facts.RestartError = k.restartErr
	return facts
}

// restartT3 runs `t3 service restart` for the runner's T3 Code home. On Linux the runner is a system service with no
// login session, so it reaches the person's user manager, which T3 Code's service lives in, at its runtime directory.
func (c *Client) restartT3(ctx context.Context) error {
	t3 := c.findT3(c.cfg.T3Home)
	if t3 == "" {
		return errors.New("T3 Code's t3 command is not installed on this computer")
	}
	env := append(os.Environ(), "T3CODE_HOME="+c.cfg.T3Home)
	if runtime.GOOS == "linux" {
		dir := os.Getenv("XDG_RUNTIME_DIR")
		if dir == "" {
			dir = "/run/user/" + strconv.Itoa(os.Getuid())
		}
		bus := filepath.Join(dir, "bus")
		if _, err := os.Stat(bus); err != nil {
			return fmt.Errorf("this account's user service manager isn't running (no %s), so T3 Code's service can't be restarted; turn on lingering with sudo loginctl enable-linger %s", bus, accountName())
		}
		env = append(env, "XDG_RUNTIME_DIR="+dir, "DBUS_SESSION_BUS_ADDRESS=unix:path="+bus)
	}
	ctx, cancel := context.WithTimeout(ctx, t3RestartTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, t3, "service", "restart")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("t3 service restart failed (%w): %s", err, lastLine(out))
	}
	if strings.Contains(string(out), "not installed") {
		return fmt.Errorf("t3 service restart: %s", lastLine(out))
	}
	return nil
}

// accountName is the runner's own account name, for a command the person can run.
func accountName() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return "$(id -un)"
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
