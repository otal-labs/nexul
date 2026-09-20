package voice

import (
	"fmt"

	"github.com/otal-labs/nexul/internal/livekit"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// errNotConfigured is what a CredentialSource returns when no LiveKit
// connector has been saved yet, mirroring connectors.Service.ManualCredentials'
// real ErrNotFound.
var errNotConfigured = fmt.Errorf("%w: livekit manual credentials not configured", apperrs.ErrNotFound)

// livekitTestClient builds a Client for tests that only care about the
// webhook secret; WSURL/APIKey are fixed, arbitrary values.
func livekitTestClient(apiSecret string) livekit.Client {
	return livekit.Client{WSURL: "wss://lk.example.com", APIKey: "key1", APISecret: apiSecret}
}
