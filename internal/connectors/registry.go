package connectors

import "sort"

// registry maps connector id -> its static definition; OAuth is nil for unimplemented tools ("coming soon").
var registry = map[string]Connector{
	"github": {
		ID:          "github",
		Name:        "GitHub",
		Description: "Links pull requests and reviews to tickets, scans your repositories, and deploys on push",
		Category:    "development",
		Icon:        "github",
		DocsURL:     "https://docs.github.com/en/apps/oauth-apps",
		OAuth:       nil,
		GitProvider: true,
	},
	"cloudflare": {
		ID:          "cloudflare",
		Name:        "Cloudflare",
		Description: "Manages DNS records and tunnels for your deployed services",
		Category:    "infrastructure",
		Icon:        "cloudflare",
		DocsURL:     "https://dash.cloudflare.com/profile/api-tokens",
		// Cloudflare only issues OAuth clients to partners, so self-hosters connect with an API token; OAuth is wired too for instances that have one.
		Manual: []CredentialField{
			{
				Key: "api_token", Label: "API token", Secret: true,
				Hint: "Create a token at Cloudflare with the permissions below, scoped to the zone you will use.",
			},
		},
		Checks: []CredentialCheck{
			{Key: "token", Label: "Token is active", Why: "Cloudflare recognises the token and it has not expired or been revoked."},
			{Key: "zone_read", Label: "Zone → Zone: Read", Why: "Lists your zones so you can pick one, and finds the account the tunnel lives in."},
			{Key: "dns_edit", Label: "Zone → DNS: Edit", Why: "Creates and updates the records that point your hostnames at this instance."},
			{Key: "tunnel_edit", Label: "Account → Cloudflare Tunnel: Edit", Why: "Creates the tunnel and issues the token cloudflared runs with."},
			{Key: "access_apps_edit", Label: "Account → Access: Apps and Policies: Edit", Why: "Closes each paired computer's hostname to everything but this instance. Needs Zero Trust enabled once on the account."},
			{Key: "access_tokens_edit", Label: "Account → Access: Service Tokens: Edit", Why: "Issues the one service token this instance uses to pass those Access rules."},
		},
		OAuth: nil,
	},
	// Manual credential, not OAuth: LiveKit's config is just URL + API key + secret, bring-your-own Cloud or self-hosted.
	"livekit": {
		ID:          "livekit",
		Name:        "LiveKit",
		Description: "Powers voice channels with screen share and camera",
		Category:    "communication",
		Icon:        "livekit",
		Manual: []CredentialField{
			{Key: "ws_url", Label: "WebSocket URL"},
			{Key: "api_key", Label: "API key"},
			{Key: "api_secret", Label: "API secret", Secret: true},
		},
	},
}

// Registry returns every connector, sorted by ID for a stable order.
func Registry() []Connector {
	out := make([]Connector, 0, len(registry))
	for _, c := range registry {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
