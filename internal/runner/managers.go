package runner

import "encoding/json"

const maxReportedManagers = 8

// encodeManagers is the connect query's value; empty when there is nothing to report.
func encodeManagers(states []ManagerState) string {
	if len(states) == 0 {
		return ""
	}
	raw, err := json.Marshal(states)
	if err != nil {
		return ""
	}
	return string(raw)
}

// parseManagers reads a connect query's managers; a malformed report, or an entry that does not name a known state,
// is dropped rather than refusing the connection.
func parseManagers(raw string) []ManagerState {
	if raw == "" {
		return nil
	}
	var in []ManagerState
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return nil
	}
	var out []ManagerState
	for _, s := range in {
		if len(out) == maxReportedManagers {
			break
		}
		known := s.State == ManagerReady || s.State == ManagerTokenRejected || s.State == ManagerUnreachable
		if s.Manager == "" || len(s.Manager) > 32 || !known {
			continue
		}
		out = append(out, s)
	}
	return out
}
