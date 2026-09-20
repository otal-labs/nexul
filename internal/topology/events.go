package topology

// TopicUpdated's consumers are web (WS push) and mcp (re-index).
const TopicUpdated = "topology.updated"

// Topics returns every topic the topology domain publishes.
func Topics() []string {
	return []string{TopicUpdated}
}

// UpdatedEvent is topology.updated's payload; field names are part of the published contract (ADR 0044), additive-only.
type UpdatedEvent struct {
	Environment string `json:"environment"`
	Canvas      Canvas `json:"canvas"`
}
