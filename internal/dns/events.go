package dns

// TopicRecordChanged is published on every record lifecycle change via the outbox.
const TopicRecordChanged = "dns.record_changed"

// TopicTunnelChanged is published on every tunnel lifecycle change via the outbox.
const TopicTunnelChanged = "dns.tunnel_changed"

// TopicGatewayChanged is published on every gateway lifecycle change via the outbox.
const TopicGatewayChanged = "dns.gateway_changed"

// TopicExposureChanged is published on every exposure lifecycle change via the outbox.
const TopicExposureChanged = "dns.exposure_changed"

// RecordChangedEvent is the record_changed payload; carries no credentials or provider internals.
type RecordChangedEvent struct {
	ZoneID   string     `json:"zone_id"`
	Zone     string     `json:"zone,omitempty"`
	RecordID string     `json:"record_id,omitempty"`
	Action   string     `json:"action"` // created | updated | deleted
	Type     RecordType `json:"type,omitempty"`
	Name     string     `json:"name,omitempty"`
	Service  string     `json:"service,omitempty"`
}

// TunnelChangedEvent is the tunnel_changed payload; carries no tunnel token or provider internals.
type TunnelChangedEvent struct {
	TunnelID string `json:"tunnel_id"`
	Name     string `json:"name,omitempty"`
	Action   string `json:"action"` // created | routed | rotated | deleted
	Hostname string `json:"hostname,omitempty"`
	Service  string `json:"service,omitempty"`
}

// GatewayChangedEvent is the dns.gateway_changed payload.
type GatewayChangedEvent struct {
	GatewayID     string      `json:"gateway_id"`
	Kind          GatewayKind `json:"kind,omitempty"`
	DockerNetwork string      `json:"docker_network,omitempty"`
	Action        string      `json:"action"` // created | deleted
}

// Topics returns every topic the dns domain publishes.
func Topics() []string {
	return []string{TopicRecordChanged, TopicTunnelChanged, TopicGatewayChanged, TopicExposureChanged}
}

// ExposureChangedEvent is the dns.exposure_changed payload.
type ExposureChangedEvent struct {
	ExposureID string `json:"exposure_id"`
	GatewayID  string `json:"gateway_id"`
	Hostname   string `json:"hostname,omitempty"`
	Service    string `json:"service,omitempty"`
	Port       int    `json:"port,omitempty"`
	Action     string `json:"action"` // created | deleted
}
