package sso

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicChanged is published when the instance's single sign-on settings are saved or removed.
const TopicChanged = "instance.sso_changed"

// TopicSAMLChanged is published when the SAML provider is saved, removed or its certificates change.
const TopicSAMLChanged = "instance.saml_changed"

// TopicExpectedChanged is published when an expected sign-in is saved, removed, or used by the first sign-in it names.
const TopicExpectedChanged = "instance.sso_expected_changed"

// Topics lists every topic this package publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicChanged, Payload: ChangedEvent{}}, {Name: TopicSAMLChanged, Payload: SAMLChangedEvent{}},
		{Name: TopicExpectedChanged, Payload: ExpectedChangedEvent{}},
	}
}

// ExpectedChangedEvent is the instance.sso_expected_changed payload.
type ExpectedChangedEvent struct {
	AccountID string    `json:"account_id" jsonschema:"The account the expected sign-in lands on."`
	Email     string    `json:"email" jsonschema:"The email the first sign-in through the provider brings."`
	Removed   bool      `json:"removed" jsonschema:"True when the expected sign-in was removed without being used."`
	Used      bool      `json:"used" jsonschema:"True when a first sign-in used it, which deletes it."`
	ChangedBy string    `json:"changed_by" jsonschema:"The account that saved or removed it; the account signing in when used."`
	ChangedAt time.Time `json:"changed_at"`
}

// ChangedEvent is the instance.sso_changed payload; the client secret stays out.
type ChangedEvent struct {
	DisplayName string    `json:"display_name" jsonschema:"The provider's name on the sign-in button; empty when the settings were removed."`
	IssuerURL   string    `json:"issuer_url" jsonschema:"The provider's issuer URL; empty when the settings were removed."`
	Only        bool      `json:"only" jsonschema:"True while single sign-on is the only way in besides the instance Owner's passkey."`
	Removed     bool      `json:"removed" jsonschema:"True when the settings were removed and the instance holds none."`
	ChangedBy   string    `json:"changed_by" jsonschema:"The account that saved or removed the settings."`
	ChangedAt   time.Time `json:"changed_at"`
}

// SAMLChangedEvent is the instance.saml_changed payload; no key or certificate is in it.
type SAMLChangedEvent struct {
	DisplayName string    `json:"display_name" jsonschema:"The provider's name on the sign-in button; empty when the settings were removed."`
	EntityID    string    `json:"entity_id" jsonschema:"The identity provider's entity id from its metadata; empty when the settings were removed."`
	Only        bool      `json:"only" jsonschema:"True while single sign-on is the only way in besides the instance Owner's passkey."`
	Removed     bool      `json:"removed" jsonschema:"True when the settings were removed and the instance holds none."`
	ChangedBy   string    `json:"changed_by" jsonschema:"The account that made the change."`
	ChangedAt   time.Time `json:"changed_at"`
}
