package t3client

import (
	"encoding/json"
	"fmt"

	"github.com/otal-labs/nexul/internal/harness"
)

// Providers reads usable provider instances from the handshake's config, no extra RPC; legacy models are dropped.
func (c *Client) Providers() ([]harness.Provider, error) {
	var config struct {
		Providers []struct {
			InstanceID   string `json:"instanceId"`
			Driver       string `json:"driver"`
			DisplayName  string `json:"displayName"`
			Enabled      bool   `json:"enabled"`
			Installed    bool   `json:"installed"`
			Availability string `json:"availability"`
			Models       []struct {
				Slug      string `json:"slug"`
				Name      string `json:"name"`
				IsDefault bool   `json:"isDefault"`
				IsLegacy  bool   `json:"isLegacy"`
			} `json:"models"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(c.config, &config); err != nil {
		return nil, fmt.Errorf("decode server config providers: %w", err)
	}
	providers := make([]harness.Provider, 0, len(config.Providers))
	for _, p := range config.Providers {
		if p.InstanceID == "" || !p.Enabled || !p.Installed || p.Availability == "unavailable" {
			continue
		}
		name := p.DisplayName
		if name == "" {
			name = p.Driver
		}
		models := make([]harness.ProviderModel, 0, len(p.Models))
		for _, m := range p.Models {
			if m.Slug == "" || m.IsLegacy {
				continue
			}
			models = append(models, harness.ProviderModel{Slug: m.Slug, Name: m.Name, IsDefault: m.IsDefault})
		}
		providers = append(providers, harness.Provider{ID: p.InstanceID, Name: name, Models: models})
	}
	return providers, nil
}
