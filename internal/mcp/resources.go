package mcp

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
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
