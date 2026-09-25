package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
)

// resource is one readable entity a user can attach to a conversation; a URI containing "{id}" is a template.
type resource struct {
	uri, name, title, description, mimeType string
	read                                    func(ctx context.Context, id string) (string, error)
}

func (r resource) register(srv *sdk.Server, logger *slog.Logger) {
	h := r.handler(logger)
	if strings.Contains(r.uri, "{id}") {
		srv.AddResourceTemplate(&sdk.ResourceTemplate{
			URITemplate: r.uri, Name: r.name, Title: r.title, Description: r.description, MIMEType: r.mimeType,
		}, h)
		return
	}
	srv.AddResource(&sdk.Resource{URI: r.uri, Name: r.name, Title: r.title, Description: r.description, MIMEType: r.mimeType}, h)
}

// handler reads live content, so the result is never cached; a missing or unreadable entity is not found, never leaked.
func (r resource) handler(logger *slog.Logger) sdk.ResourceHandler {
	prefix, _, _ := strings.Cut(r.uri, "{id}")
	return func(ctx context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
		uri := req.Params.URI
		id := strings.TrimPrefix(uri, prefix)
		text, err := r.read(ctx, id)
		if errors.Is(err, apperrs.ErrNotFound) || errors.Is(err, apperrs.ErrForbidden) || errors.Is(err, apperrs.ErrUnauthorized) {
			return nil, sdk.ResourceNotFoundError(uri)
		}
		if err != nil {
			traceID := logging.TraceIDFromCtx(ctx)
			if traceID == "" {
				traceID = logging.NewTraceID()
			}
			logger.Error("mcp resource read failed", "trace_id", traceID, "uri", uri, "err", err)
			return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: internalMessage(traceID)}
		}
		return &sdk.ReadResourceResult{
			Cacheable: sdk.Cacheable{TTLMs: 0, CacheScope: "private"},
			Contents:  []*sdk.ResourceContents{{URI: uri, MIMEType: r.mimeType, Text: text}},
		}, nil
	}
}

func docResource(s *docs.Service) resource {
	return resource{
		uri: "docs://{id}", name: "doc", title: "Doc", mimeType: "text/markdown",
		description: "A doc's title and full body as markdown, the same content doc_get returns.",
		read: func(ctx context.Context, id string) (string, error) {
			d, err := s.Get(ctx, id)
			if err != nil {
				return "", err
			}
			md, err := richtext.ToMarkdown(d.Body)
			if err != nil {
				return "", fmt.Errorf("render doc %s: %w", id, err)
			}
			return "# " + d.Title + "\n\n" + md, nil
		},
	}
}

func ticketResource(s *tickets.Service) resource {
	return resource{
		uri: "tickets://{id}", name: "ticket", title: "Ticket", mimeType: "text/markdown",
		description: "A ticket's title, status, and body as markdown; ticket_get returns the full record.",
		read: func(ctx context.Context, id string) (string, error) {
			t, err := s.Get(ctx, id)
			if err != nil {
				return "", err
			}
			return "# " + t.Title + "\n\nstatus: " + string(t.Status) + "\n\n" + t.Body, nil
		},
	}
}

func topologyResource(s *topology.Service) resource {
	return resource{
		uri: "topology://current", name: "topology", title: "Current topology", mimeType: "application/json",
		description: "The topology canvas of the default environment, the same content topology_get returns.",
		read: func(ctx context.Context, _ string) (string, error) {
			c, err := s.Get(ctx, topology.DefaultEnvironment)
			if err != nil {
				return "", err
			}
			b, err := json.Marshal(c)
			if err != nil {
				return "", fmt.Errorf("marshal topology: %w", err)
			}
			return string(b), nil
		},
	}
}
