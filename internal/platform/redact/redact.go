// Package redact hides credentials in text that is saved, such as an agent turn's transcript.
package redact

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
)

// Placeholder replaces every token Tokens finds.
const Placeholder = "[redacted token]"

// bearerToken matches auth's dep_ and ses_ prefixes plus their base64url body; TestTokens_HidesAMintedToken in auth pins the shape.
var bearerToken = regexp.MustCompile(`(?:dep|ses)_[A-Za-z0-9_-]{43,}`)

// queryToken matches a token= query value, how every WebSocket carries its credential whatever its prefix.
var queryToken = regexp.MustCompile(`([?&]token=)[^&#\s"'\\]+`)

// Tokens replaces every personal access or session token, and every token= query value, in s with Placeholder.
func Tokens(s string) string {
	s = queryToken.ReplaceAllString(s, "${1}"+Placeholder)
	return bearerToken.ReplaceAllString(s, Placeholder)
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
