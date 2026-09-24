// Package redact hides credentials in text that is saved, such as an agent turn's transcript.
package redact

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
)

// Placeholder replaces every personal access token Tokens finds.
const Placeholder = "[redacted token]"

// personalAccessToken matches auth's dep_ prefix plus its base64url body; TestTokens_HidesAMintedToken in auth pins the shape.
var personalAccessToken = regexp.MustCompile(`dep_[A-Za-z0-9_-]{43,}`)

// Tokens replaces every personal access token in s with Placeholder.
func Tokens(s string) string {
	return personalAccessToken.ReplaceAllString(s, Placeholder)
}

// JSON marshals v with every personal access token replaced and reports whether it found one; base64url needs no escaping.
func JSON(v any) (json.RawMessage, bool, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, false, fmt.Errorf("marshal for redaction: %w", err)
	}
	redacted := Tokens(string(raw))
	return json.RawMessage(redacted), redacted != string(raw), nil
}

// Publisher is the live-push seam the domains each declare; Live wraps one.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

// Live redacts every frame before it leaves the server, since live frames reach everyone who can see the thread.
type Live struct {
	Publisher
}

// Publish sends payload unchanged when it holds no token, so the common frame keeps its type.
func (l Live) Publish(ctx context.Context, topic string, payload any) error {
	raw, found, err := JSON(payload)
	if err != nil {
		return err
	}
	if !found {
		return l.Publisher.Publish(ctx, topic, payload)
	}
	return l.Publisher.Publish(ctx, topic, raw)
}
