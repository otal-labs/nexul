// Package push sends one Expo push message per phone session for every notification written to an inbox.
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// ExpoEndpoint is Expo's push API; tests point Config.Endpoint at a fake server instead.
const ExpoEndpoint = "https://exp.host/--/api/v2/push/send"

// batchSize is Expo's documented cap on messages per request.
const batchSize = 100

const requestTimeout = 15 * time.Second

// Target is one phone session holding a push token.
type Target struct {
	SessionID string
	UserID    string
	Token     string
}

// TokenStore reads and clears push tokens without push importing auth (ADR 0017).
type TokenStore interface {
	ListPushTargets(ctx context.Context, userIDs []string) ([]Target, error)
	ClearPushToken(ctx context.Context, sessionID, userID string) error
}

// WorkspaceNamer resolves a workspace's display name for the message body.
type WorkspaceNamer interface {
	WorkspaceName(ctx context.Context, workspaceID string) (string, error)
}

// InstanceURLGate answers the instance URL the phone dials back to for the notification's content.
type InstanceURLGate interface {
	GetInstanceURL(ctx context.Context) (string, error)
}

// Config wires the sender; Endpoint and Client are injectable so tests never reach Expo.
type Config struct {
	Tokens     TokenStore
	Workspaces WorkspaceNamer
	Instance   InstanceURLGate
	Endpoint   string
	Client     *http.Client
	Logger     *slog.Logger
}

// Sender consumes notification.push_requested and posts to Expo in batches.
type Sender struct {
	cfg Config
}

// New builds a Sender, defaulting the endpoint and client.
func New(cfg Config) *Sender {
	if cfg.Endpoint == "" {
		cfg.Endpoint = ExpoEndpoint
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: requestTimeout}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Sender{cfg: cfg}
}

// pushRequestedEvent mirrors workspace.NotificationPushRequestedEvent, declared consumer-side (ADR 0017).
type pushRequestedEvent struct {
	Notifications []pushItem `json:"notifications"`
}

type pushItem struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
}

// message is one Expo push message; data is id-only so nothing sensitive passes through Expo.
type message struct {
	To    string `json:"to"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Data  struct {
		NotificationID string `json:"notification_id"`
		Host           string `json:"host"`
	} `json:"data"`
	target Target
}

// ticket is Expo's per-message result, positional with the request.
type ticket struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details struct {
		Error string `json:"error"`
	} `json:"details"`
}

// HandleNotificationPushRequested fans each notification out to its user's phones; Expo's per-message
// errors are logged, a transport failure returns so the bus applies its own retry and dead-letter path.
func (s *Sender) HandleNotificationPushRequested(ctx context.Context, ev eventbus.Event) error {
	var e pushRequestedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse notification.push_requested: %w", err))
	}
	msgs, err := s.messages(ctx, e.Notifications)
	if err != nil {
		return err
	}
	for start := 0; start < len(msgs); start += batchSize {
		end := min(start+batchSize, len(msgs))
		if err := s.send(ctx, msgs[start:end]); err != nil {
			return apperrs.Retryable(err)
		}
	}
	return nil
}

func (s *Sender) messages(ctx context.Context, items []pushItem) ([]message, error) {
	userIDs := make([]string, 0, len(items))
	for _, it := range items {
		if it.UserID != "" {
			userIDs = append(userIDs, it.UserID)
		}
	}
	if len(userIDs) == 0 {
		return nil, nil
	}
	targets, err := s.cfg.Tokens.ListPushTargets(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("list push targets: %w", err)
	}
	if len(targets) == 0 {
		return nil, nil
	}
	byUser := map[string][]Target{}
	for _, t := range targets {
		byUser[t.UserID] = append(byUser[t.UserID], t)
	}
	host := s.host(ctx)
	names := map[string]string{}
	var msgs []message
	for _, it := range items {
		for _, t := range byUser[it.UserID] {
			m := message{To: t.Token, Title: "Nexul", Body: "New activity in " + s.workspaceName(ctx, names, it.WorkspaceID), target: t}
			m.Data.NotificationID = it.ID
			m.Data.Host = host
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func (s *Sender) host(ctx context.Context) string {
	if s.cfg.Instance == nil {
		return ""
	}
	host, err := s.cfg.Instance.GetInstanceURL(ctx)
	if err != nil {
		s.cfg.Logger.Warn("push: instance url", "error", err)
		return ""
	}
	return strings.TrimSuffix(host, "/")
}

// workspaceName memoises per event; a notification without a workspace, or one whose lookup fails, reads "your workspace".
func (s *Sender) workspaceName(ctx context.Context, cache map[string]string, workspaceID string) string {
	if workspaceID == "" || s.cfg.Workspaces == nil {
		return "your workspace"
	}
	if name, ok := cache[workspaceID]; ok {
		return name
	}
	name, err := s.cfg.Workspaces.WorkspaceName(ctx, workspaceID)
	if err != nil || strings.TrimSpace(name) == "" {
		name = "your workspace"
	}
	cache[workspaceID] = name
	return name
}

func (s *Sender) send(ctx context.Context, msgs []message) (err error) {
	body, err := json.Marshal(msgs)
	if err != nil {
		return fmt.Errorf("encode push batch: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := s.cfg.Client.Do(req)
	if err != nil {
		return fmt.Errorf("post push batch: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("post push batch: expo answered %d", resp.StatusCode)
	}
	var out struct {
		Data []ticket `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode push tickets: %w", err)
	}
	s.settle(ctx, msgs, out.Data)
	return nil
}

// settle walks Expo's positional tickets: a DeviceNotRegistered token is cleared, any other error is logged.
func (s *Sender) settle(ctx context.Context, msgs []message, tickets []ticket) {
	log := logging.FromCtx(ctx)
	for i, tk := range tickets {
		if i >= len(msgs) || tk.Status == "ok" {
			continue
		}
		t := msgs[i].target
		if tk.Details.Error != "DeviceNotRegistered" {
			log.Warn("push: expo rejected message", "session_id", t.SessionID, "error", tk.Details.Error, "message", tk.Message)
			continue
		}
		if err := s.cfg.Tokens.ClearPushToken(ctx, t.SessionID, t.UserID); err != nil {
			log.Warn("push: clear unregistered token", "session_id", t.SessionID, "error", err)
		}
	}
}
