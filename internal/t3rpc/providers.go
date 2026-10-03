package t3rpc

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type configProvider struct {
	InstanceID   string        `json:"instanceId"`
	Driver       string        `json:"driver"`
	DisplayName  string        `json:"displayName"`
	Enabled      bool          `json:"enabled"`
	Installed    bool          `json:"installed"`
	Availability string        `json:"availability"`
	Models       []configModel `json:"models"`
}

type configModel struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	SubProvider  string `json:"subProvider"`
	Badge        string `json:"badge"`
	IsDefault    bool   `json:"isDefault"`
	IsLegacy     bool   `json:"isLegacy"`
	Capabilities *struct {
		OptionDescriptors []optionDescriptor `json:"optionDescriptors"`
	} `json:"capabilities"`
}

type optionChoice struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	IsDefault   bool   `json:"isDefault"`
}

type optionDescriptor struct {
	ID                   string         `json:"id"`
	Label                string         `json:"label"`
	Description          string         `json:"description"`
	Type                 string         `json:"type"`
	Options              []optionChoice `json:"options"`
	CurrentValue         any            `json:"currentValue"`
	PromptInjectedValues []string       `json:"promptInjectedValues"`
}

// Providers reads usable provider instances from the handshake's config, no extra RPC.
func (c *Conn) Providers() ([]harness.Provider, error) {
	var config struct {
		Providers []configProvider `json:"providers"`
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
			if m.Slug == "" {
				continue
			}
			models = append(models, providerModel(m))
		}
		providers = append(providers, harness.Provider{ID: p.InstanceID, Driver: p.Driver, Name: name, Models: models})
	}
	return providers, nil
}

func providerModel(m configModel) harness.ProviderModel {
	model := harness.ProviderModel{
		Slug: m.Slug, Name: m.Name, IsDefault: m.IsDefault, SubProvider: m.SubProvider, IsNew: m.Badge == "new", IsLegacy: m.IsLegacy,
	}
	if m.Capabilities == nil {
		return model
	}
	for _, d := range m.Capabilities.OptionDescriptors {
		if option, ok := modelOption(d); ok {
			model.Options = append(model.Options, option)
		}
	}
	return model
}

// modelOption maps one descriptor; a choice T3 applies by rewriting the prompt (ultrathink) is dropped, since it only works from T3's composer.
func modelOption(d optionDescriptor) (harness.ModelOption, bool) {
	option := harness.ModelOption{ID: d.ID, Label: d.Label, Description: d.Description}
	if d.Type == string(harness.OptionSwitch) {
		option.Type = harness.OptionSwitch
		option.DefaultOn = d.CurrentValue == true
		return option, d.ID != ""
	}
	option.Type = harness.OptionSelect
	current, _ := d.CurrentValue.(string)
	marked := slices.ContainsFunc(d.Options, func(o optionChoice) bool { return o.IsDefault })
	for _, o := range d.Options {
		if slices.Contains(d.PromptInjectedValues, o.ID) {
			continue
		}
		option.Choices = append(option.Choices, harness.OptionChoice{
			ID: o.ID, Label: o.Label, Description: o.Description, IsDefault: o.IsDefault || (!marked && o.ID == current),
		})
	}
	return option, d.ID != "" && len(option.Choices) > 0
}

// DefaultModel picks providerID's default model, falling back to its first current one, so an empty model never
// reaches T3, which rejects an empty modelSelection.model as a defect.
func DefaultModel(providers []harness.Provider, providerID string) (string, error) {
	for _, p := range providers {
		if p.ID != providerID {
			continue
		}
		for _, m := range p.Models {
			if m.IsDefault {
				return m.Slug, nil
			}
		}
		for _, m := range p.Models {
			if !m.IsLegacy {
				return m.Slug, nil
			}
		}
		return "", fmt.Errorf("%w: provider %s has no models", apperrs.ErrInvalid, providerID)
	}
	return "", fmt.Errorf("%w: provider %s not found", apperrs.ErrInvalid, providerID)
}
