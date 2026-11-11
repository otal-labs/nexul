// Package onepassword resolves op:// references through the 1Password SDK. Only server/cmd and runner/cmd may
// import it: the SDK embeds a 9.5 MB wasm core that cmd/nexul must not carry.
package onepassword

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	sdk "github.com/1password/onepassword-sdk-go"

	"github.com/otal-labs/nexul/internal/platform/secretref"
)

const integrationName = "nexul"

// Resolver holds the one SDK client for the life of the process; creating it costs about 2.6 s of CPU.
type Resolver struct {
	secrets sdk.SecretsAPI
}

// New creates the client with a service account token. A failure with no network is ErrUnreachable.
func New(ctx context.Context, token, version string) (*Resolver, error) {
	client, err := sdk.NewClient(ctx,
		sdk.WithServiceAccountToken(token),
		sdk.WithIntegrationInfo(integrationName, version),
	)
	if err != nil {
		return nil, fmt.Errorf("create 1password client: %w", classify(ctx, err))
	}
	return &Resolver{secrets: client.Secrets()}, nil
}

func (r *Resolver) Scheme() string { return "op" }

// ResolveAll is one SDK call, one read against the token's hourly limit however many refs it holds.
func (r *Resolver) ResolveAll(ctx context.Context, refs []string) ([]secretref.Result, error) {
	resp, err := r.secrets.ResolveAll(ctx, refs)
	if err != nil {
		return nil, fmt.Errorf("resolve 1password references: %w", classify(ctx, err))
	}
	out := make([]secretref.Result, len(refs))
	for i, ref := range refs {
		out[i] = mapResponse(resp.IndividualResponses[ref])
	}
	return out, nil
}

func mapResponse(r sdk.Response[sdk.ResolvedReference, sdk.ResolveReferenceError]) secretref.Result {
	if r.Error != nil {
		return secretref.Result{Err: mapRefError(r.Error.Type)}
	}
	if r.Content == nil {
		return secretref.Result{Err: secretref.ErrFailed}
	}
	return secretref.Result{Value: r.Content.Secret}
}

// mapRefError names the SDK's error type only, never anything resolved.
func mapRefError(t sdk.ResolveReferenceErrorTypes) error {
	switch t {
	case sdk.ResolveReferenceErrorTypeVariantItemNotFound:
		return secretref.ErrNotFound
	case sdk.ResolveReferenceErrorTypeVariantFieldNotFound:
		return secretref.ErrFieldMissing
	case sdk.ResolveReferenceErrorTypeVariantVaultNotFound:
		return secretref.ErrNoAccess
	}
	return fmt.Errorf("%w: %s", secretref.ErrFailed, t)
}

// classify marks a transport failure ErrUnreachable; the SDK reports a dial error as plain text, so it is matched by wording.
func classify(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return err
	}
	var netErr net.Error
	msg := err.Error()
	if errors.As(err, &netErr) || strings.Contains(msg, "dial") || strings.Contains(msg, "no such host") || strings.Contains(msg, "connection refused") {
		return fmt.Errorf("%w: %w", secretref.ErrUnreachable, err)
	}
	return err
}
