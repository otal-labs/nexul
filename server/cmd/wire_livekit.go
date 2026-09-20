package main

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/livekit"
)

// livekitVerifier adapts livekit's client to connectors' Verifier seam: confirms fields work with one ListRooms call.
type livekitVerifier struct{}

func (livekitVerifier) Verify(ctx context.Context, fields map[string]string) error {
	client := &livekit.Client{WSURL: fields["ws_url"], APIKey: fields["api_key"], APISecret: fields["api_secret"]}
	if _, err := client.ListRooms(ctx); err != nil {
		return fmt.Errorf("list rooms: %w", err)
	}
	return nil
}
