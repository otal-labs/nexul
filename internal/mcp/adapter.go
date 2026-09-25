// Package mcp is the Model Context Protocol adapter: the official SDK's stateless Streamable HTTP handler over the
// tools, resources, and prompts the domains declare (ADR 0067).
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/version"
)

// listTTL is how long a client may reuse tools/list and the other lists: fixed per process, so only an upgrade changes them.
const listTTL = 5 * time.Minute

// Per-actor tool-call budget: generous for an agent's parallel reads, tight enough that a loop cannot starve the writer.
const (
	callsPerSecond = 10
	callBurst      = 40
)

// server is the adapter's content: what it serves, independent of which domains supplied it.
type server struct {
	tools        []mcptool.Tool
	resources    []resource
	prompts      []prompt
	instructions string
	logger       *slog.Logger
}

// handler builds the SDK server and wraps its stateless handler in the origin check.
func (s server) handler(instanceURL func(context.Context) (string, error)) http.Handler {
	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}
	srv := sdk.NewServer(&sdk.Implementation{Name: "nexul", Title: "Nexul", Version: version.Version}, &sdk.ServerOptions{
		Instructions: s.instructions,
		Capabilities: &sdk.ServerCapabilities{
			Tools:     &sdk.ToolCapabilities{},
			Resources: &sdk.ResourceCapabilities{},
			Prompts:   &sdk.PromptCapabilities{},
		},
		SetCacheable: setCacheable,
	})
	limits := &limiters{byActor: map[string]*rate.Limiter{}}
	for _, t := range s.tools {
		srv.AddTool(sdkTool(t), toolHandler(t, limits, logger))
	}
	for _, r := range s.resources {
		r.register(srv, logger)
	}
	for _, p := range s.prompts {
		srv.AddPrompt(p.sdkPrompt(), p.handler())
	}
	h := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv }, &sdk.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		// Bearer auth on every request makes DNS rebinding useless, and the guard would 403 a same-host reverse proxy.
		DisableLocalhostProtection: true,
	})
	return originGuard(instanceURL, h)
}

func sdkTool(t mcptool.Tool) *sdk.Tool {
	// A read changes nothing, so it is neither destructive nor at risk from a repeat, whatever the other hints say.
	destructive, openWorld := !t.Hints.ReadOnly && !t.Hints.Additive, !t.Hints.Local
	return &sdk.Tool{
		Name:        t.Name,
		Title:       t.Title,
		Description: t.Description,
		InputSchema: t.InputSchema,
		Annotations: &sdk.ToolAnnotations{
			Title:           t.Title,
			ReadOnlyHint:    t.Hints.ReadOnly,
			DestructiveHint: &destructive,
			IdempotentHint:  t.Hints.Idempotent || t.Hints.ReadOnly,
			OpenWorldHint:   &openWorld,
		},
	}
}

// toolHandler runs a tool as the caller and turns every failure into an isError result the model can read.
func toolHandler(t mcptool.Tool, limits *limiters, logger *slog.Logger) sdk.ToolHandler {
	return func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		traceID := logging.TraceIDFromCtx(ctx)
		if traceID == "" {
			traceID = logging.NewTraceID()
			ctx = logging.CtxWithTraceID(ctx, traceID)
		}
		actor, _ := identity.ActorFromCtx(ctx)
		log := logger.With("trace_id", traceID, "tool", t.Name, "actor", actor.ID)
		if wait := limits.wait(actor.ID); wait > 0 {
			log.Warn("mcp tool call rate limited")
			return errorResult(fmt.Sprintf("Too many tool calls; retry in %s.", wait.Round(time.Second))), nil
		}
		start := time.Now()
		out, err := t.Call(logging.CtxWithLogger(ctx, log), req.Params.Arguments)
		if err != nil {
			message, internal := toolErrorMessage(err, traceID)
			logAt := log.Info
			if internal {
				logAt = log.Error
			}
			logAt("mcp tool call failed", "duration", time.Since(start), "err", err)
			return errorResult(message), nil
		}
		text, err := resultText(out)
		if err != nil {
			log.Error("mcp tool result not serializable", "err", err)
			return errorResult(internalMessage(traceID)), nil
		}
		log.Info("mcp tool call", "duration", time.Since(start))
		return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}}, nil
	}
}

func errorResult(message string) *sdk.CallToolResult {
	return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: message}}}
}

// toolErrorMessage passes a known sentinel's message through and hides everything else, as the HTTP gateway does.
func toolErrorMessage(err error, traceID string) (message string, internal bool) {
	var partial *mcptool.PartialError
	if errors.As(err, &partial) {
		message, internal = toolErrorMessage(partial.Err, traceID)
		return message + "; already applied: " + strings.Join(partial.Applied, ", "), internal
	}
	for _, known := range []error{apperrs.ErrInvalid, apperrs.ErrNotFound, apperrs.ErrConflict, apperrs.ErrForbidden, apperrs.ErrUnauthorized} {
		if errors.Is(err, known) {
			return err.Error(), false
		}
	}
	return internalMessage(traceID), true
}

func internalMessage(traceID string) string {
	return "internal error (trace " + traceID + ")"
}

func resultText(out any) (string, error) {
	if s, ok := out.(string); ok {
		return s, nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// setCacheable gives the fixed lists and discovery a short public TTL; resource reads set their own zero TTL.
func setCacheable(_ context.Context, req sdk.Request, c *sdk.Cacheable) {
	if _, ok := req.(*sdk.ReadResourceRequest); ok {
		return
	}
	c.TTLMs = int(listTTL / time.Millisecond)
	c.CacheScope = "public"
}

// limiters holds one token bucket per actor; single-tenant, so the map is bounded by the instance's accounts.
type limiters struct {
	mu      sync.Mutex
	byActor map[string]*rate.Limiter
}

// wait returns zero when the call may run now, otherwise how long until it could.
func (l *limiters) wait(actorID string) time.Duration {
	l.mu.Lock()
	lim, ok := l.byActor[actorID]
	if !ok {
		lim = rate.NewLimiter(callsPerSecond, callBurst)
		l.byActor[actorID] = lim
	}
	l.mu.Unlock()
	r := lim.Reserve()
	delay := r.Delay()
	if delay == 0 {
		return 0
	}
	r.Cancel()
	return max(delay, time.Second)
}
