package automations

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// DialinConfig wires the server-side automations WS handler (ADR 0046).
type DialinConfig struct {
	// Repo reads the automation record for delivery decisions; authentication itself goes through Service.
	Repo Repo
	// Cursors persists each automation's delivery position.
	Cursors CursorRepo
	// EventLog is the durable event source deliveries replay from.
	EventLog EventLogReader
	// Runs persists run reports.
	Runs RunsRepo
	// Secrets is pushed to the automation at connect; defaults to NoopSecretsProvider.
	Secrets SecretsProvider
	// Logger for lifecycle and failure logs; defaults to slog.Default().
	Logger *slog.Logger
	// PollInterval is how often an idle connection checks for new matching events; defaults to 500ms.
	PollInterval time.Duration
	// BatchSize bounds one poll's catch-up read; defaults to 50.
	BatchSize int
	// WriteTimeout bounds each frame write; defaults to 10s.
	WriteTimeout time.Duration
	// ReadLimit caps a single inbound frame; defaults to 2MiB.
	ReadLimit int64
}

// DialinHandler is the server-side WS endpoint automations dial into (ADR 0046); a new connection replaces the old one.
type DialinHandler struct {
	log *slog.Logger
	svc *Service
	cfg DialinConfig

	mu    sync.Mutex
	conns map[string]*automationConn
}

// NewDialinHandler wires the server-side automations dial-in handler.
func NewDialinHandler(svc *Service, cfg DialinConfig) *DialinHandler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.ReadLimit <= 0 {
		cfg.ReadLimit = 2 << 20
	}
	if cfg.Secrets == nil {
		cfg.Secrets = NoopSecretsProvider{}
	}
	return &DialinHandler{log: cfg.Logger, svc: svc, cfg: cfg, conns: make(map[string]*automationConn)}
}

// automationConn is one live dial-in connection: at most one run in flight, so delivery is strictly sequential.
type automationConn struct {
	id string // automation id
	ws *websocket.Conn

	writeMu sync.Mutex

	runMu  sync.Mutex
	active *activeRun
}

// activeRun tracks the run in flight on a connection until a terminal frame or a dead connection resolves it.
type activeRun struct {
	id         string
	eventTopic string
	eventID    string
	cursor     Cursor // the log position this event occupies
	startedAt  time.Time
	logs       string
	done       chan struct{}
}

// Disconnect force-drops a live connection so a revoked token stops working immediately, even mid-connection.
func (h *DialinHandler) Disconnect(automationID, reason string) {
	h.mu.Lock()
	c, ok := h.conns[automationID]
	h.mu.Unlock()
	if !ok {
		return
	}
	h.log.Info("automation connection force-closed", "automation_id", automationID, "reason", reason)
	_ = c.ws.CloseNow()
}

// CloseAll drops every live connection, e.g. on server shutdown.
func (h *DialinHandler) CloseAll(reason string) {
	h.mu.Lock()
	conns := make([]*automationConn, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		h.log.Info("automation connection force-closed", "automation_id", c.id, "reason", reason)
		_ = c.ws.CloseNow()
	}
}

// ServeHTTP authenticates and upgrades an automation connection, then owns it until it disconnects.
func (h *DialinHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	automation, err := h.svc.AuthenticateToken(r.Context(), token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.log.Warn("automation websocket upgrade failed", "error", err)
		return
	}
	conn.SetReadLimit(h.cfg.ReadLimit)
	h.handleConn(r.Context(), automation.ID, conn)
}

func (h *DialinHandler) handleConn(ctx context.Context, automationID string, ws *websocket.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c := &automationConn{id: automationID, ws: ws}

	h.mu.Lock()
	if old, ok := h.conns[automationID]; ok {
		_ = old.ws.CloseNow()
	}
	h.conns[automationID] = c
	h.mu.Unlock()
	defer h.removeConn(c)

	h.log.Info("automation connected", "automation_id", automationID)

	// The wire protocol opens with an announce frame; anything else is refused outright.
	var raw json.RawMessage
	if err := wsjson.Read(ctx, ws, &raw); err != nil {
		h.log.Warn("automation announce read failed", "automation_id", automationID, "error", err)
		return
	}
	frame, err := ParseFrame(raw)
	if err != nil || frame.Type != FrameAnnounce {
		h.log.Warn("automation did not announce first", "automation_id", automationID, "error", err)
		_ = ws.Close(websocket.StatusPolicyViolation, "expected announce frame")
		return
	}
	automation, err := h.svc.SyncFromCode(ctx, automationID, frame.Name, frame.Description, frame.Subscriptions, frame.ConfigSchema)
	if err != nil {
		h.log.Warn("automation sync from code failed", "automation_id", automationID, "error", err)
		_ = ws.Close(websocket.StatusPolicyViolation, "invalid announce")
		return
	}

	// Establish the cursor before delivery so a new automation starts at "now" instead of replaying all history.
	if err := h.ensureCursor(ctx, automationID); err != nil {
		h.log.Warn("initialize automation cursor failed", "automation_id", automationID, "error", err)
	}

	secrets, err := h.cfg.Secrets.AllSecrets(ctx)
	if err != nil {
		h.log.Warn("load workspace secrets failed", "automation_id", automationID, "error", err)
		secrets = map[string]string{}
	}
	hello := Frame{Type: FrameHello, ConfigValues: automation.ConfigValues, Secrets: secrets}
	if err := h.sendFrame(ctx, c, hello); err != nil {
		h.log.Warn("automation hello send failed", "automation_id", automationID, "error", err)
		return
	}

	go h.deliverLoop(ctx, c)
	h.readLoop(ctx, c)
}

// readLoop drains run frames until the peer closes; a run left in flight is a crash with the cursor unadvanced (ADR 0046).
func (h *DialinHandler) readLoop(ctx context.Context, c *automationConn) {
	defer func() {
		if a := c.detachActiveAny(); a != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			h.recordRun(cleanupCtx, c.id, a, RunOutcomeCrash, "connection closed", "")
			cancel()
		}
		_ = c.ws.Close(websocket.StatusNormalClosure, "")
	}()
	for {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, c.ws, &raw); err != nil {
			return
		}
		frame, err := ParseFrame(raw)
		if err != nil {
			h.log.Warn("invalid frame from automation", "automation_id", c.id, "error", err)
			_ = c.ws.Close(websocket.StatusPolicyViolation, "invalid frame")
			return
		}
		if err := h.dispatchFrame(ctx, c, *frame); err != nil {
			h.log.Warn("automation frame dispatch failed", "automation_id", c.id, "type", frame.Type, "error", err)
		}
	}
}

// dispatchFrame reports an unknown/mismatched run id, but doesn't fail the connection over it.
func (h *DialinHandler) dispatchFrame(ctx context.Context, c *automationConn, f Frame) error {
	switch f.Type {
	case FrameRunStarted:
		return c.touchActive(f.RunID)
	case FrameRunLog:
		return c.appendActiveLog(f.RunID, f.Log)
	case FrameRunFinished:
		a, err := c.detachActive(f.RunID)
		if err != nil {
			return err
		}
		h.recordRun(ctx, c.id, a, RunOutcome(f.Outcome), "", f.Log)
		return nil
	case FrameRunCrashed:
		a, err := c.detachActive(f.RunID)
		if err != nil {
			return err
		}
		h.recordRun(ctx, c.id, a, RunOutcomeCrash, f.Error, f.Log)
		return nil
	default:
		return fmt.Errorf("%w: unexpected frame type %q from automation", apperrs.ErrInvalid, f.Type)
	}
}

// recordRun persists a report; a clean finish advances the cursor, a crash keeps it so reconnect redelivers (ADR 0046).
func (h *DialinHandler) recordRun(ctx context.Context, automationID string, a *activeRun, outcome RunOutcome, errMsg, finalLog string) {
	now := time.Now().UTC()
	run := &Run{
		ID:           a.id,
		AutomationID: automationID,
		EventTopic:   a.eventTopic,
		EventID:      a.eventID,
		Outcome:      outcome,
		Error:        errMsg,
		StartedAt:    a.startedAt,
		FinishedAt:   now,
		DurationMS:   now.Sub(a.startedAt).Milliseconds(),
		Logs:         appendCappedLog(a.logs, finalLog),
		CreatedAt:    now,
	}
	if err := h.cfg.Runs.Create(ctx, run); err != nil {
		h.log.Warn("persist automation run failed", "automation_id", automationID, "run_id", a.id, "error", err)
	}
	if outcome == RunOutcomeSuccess || outcome == RunOutcomeFailure {
		if err := h.cfg.Cursors.Set(ctx, automationID, a.cursor); err != nil {
			h.log.Warn("persist automation cursor failed", "automation_id", automationID, "error", err)
		}
	}
	close(a.done)
}

// deliverLoop delivers matching events one at a time, waiting for each run's report before the next.
func (h *DialinHandler) deliverLoop(ctx context.Context, c *automationConn) {
	ticker := time.NewTicker(h.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		automation, err := h.cfg.Repo.Get(ctx, c.id)
		if err != nil {
			h.log.Warn("automation lookup failed during delivery", "automation_id", c.id, "error", err)
			continue
		}
		if !automation.Enabled || len(automation.Subscriptions) == 0 {
			continue
		}
		cursor, _, err := h.cfg.Cursors.Get(ctx, c.id)
		if err != nil {
			h.log.Warn("load automation cursor failed", "automation_id", c.id, "error", err)
			continue
		}
		events, err := h.cfg.EventLog.After(ctx, automation.Subscriptions, cursor, h.cfg.BatchSize)
		if err != nil {
			h.log.Warn("read event log failed", "automation_id", c.id, "error", err)
			continue
		}
		for _, ev := range events {
			if !h.deliverOne(ctx, c, ev) {
				return
			}
		}
	}
}

// ensureCursor writes a row even if zero, so a later connect never mistakes itself for the first (ADR 0046).
func (h *DialinHandler) ensureCursor(ctx context.Context, automationID string) error {
	_, ok, err := h.cfg.Cursors.Get(ctx, automationID)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	latest, ok, err := h.cfg.EventLog.Latest(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return h.cfg.Cursors.Set(ctx, automationID, Cursor{})
	}
	return h.cfg.Cursors.Set(ctx, automationID, latest)
}

// deliverOne sends one event, blocking until its run resolves; false means the connection is closing.
func (h *DialinHandler) deliverOne(ctx context.Context, c *automationConn, ev LogEvent) bool {
	done := make(chan struct{})
	a := &activeRun{
		id:         ids.New(),
		eventTopic: ev.Topic,
		eventID:    ev.ID,
		cursor:     Cursor{CreatedAt: ev.CreatedAt, ID: ev.ID},
		startedAt:  time.Now().UTC(),
		done:       done,
	}
	c.setActive(a)
	frame := Frame{Type: FrameEvent, RunID: a.id, EventID: ev.ID, Topic: ev.Topic, Payload: ev.Payload}
	if err := h.sendFrame(ctx, c, frame); err != nil {
		h.log.Warn("event send failed", "automation_id", c.id, "error", err)
		c.detachActiveAny()
		return false
	}
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

// sendFrame writes one frame under the write timeout; writeMu serializes the delivery loop and the hello handshake.
func (h *DialinHandler) sendFrame(ctx context.Context, c *automationConn, f Frame) error {
	ctx, cancel := context.WithTimeout(ctx, h.cfg.WriteTimeout)
	defer cancel()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return wsjson.Write(ctx, c.ws, f)
}

func (h *DialinHandler) removeConn(c *automationConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[c.id] == c {
		delete(h.conns, c.id)
	}
}

func (c *automationConn) setActive(a *activeRun) {
	c.runMu.Lock()
	c.active = a
	c.runMu.Unlock()
}

// touchActive records the automation's own reported start time for the
// named run; a run_started for anything else is a protocol violation.
func (c *automationConn) touchActive(runID string) error {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	if c.active == nil || c.active.id != runID {
		return fmt.Errorf("%w: run_started for unknown run %s", apperrs.ErrInvalid, runID)
	}
	c.active.startedAt = time.Now().UTC()
	return nil
}

func (c *automationConn) appendActiveLog(runID, chunk string) error {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	if c.active == nil || c.active.id != runID {
		return fmt.Errorf("%w: run_log for unknown run %s", apperrs.ErrInvalid, runID)
	}
	c.active.logs = appendCappedLog(c.active.logs, chunk)
	return nil
}

// detachActive clears and returns the active run if it matches runID, or an
// error if a terminal frame references anything else.
func (c *automationConn) detachActive(runID string) (*activeRun, error) {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	if c.active == nil || c.active.id != runID {
		return nil, fmt.Errorf("%w: run report for unknown run %s", apperrs.ErrInvalid, runID)
	}
	a := c.active
	c.active = nil
	return a, nil
}

// detachActiveAny clears and returns whatever run is active, used on disconnect with no report to match against.
func (c *automationConn) detachActiveAny() *activeRun {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	a := c.active
	c.active = nil
	return a
}
