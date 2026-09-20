package automations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// hostTokensPathEnv points the seeder at the file the bundled host reads for its default automations' dial-in tokens.
const hostTokensPathEnv = "NEXUL_AUTOMATIONS_HOST_TOKENS_PATH"

type hostTokenEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// writeHostToken upserts a raw token into the host token file, if configured; a failure never blocks startup (AM10).
func writeHostToken(id, name, token string) error {
	path := os.Getenv(hostTokensPathEnv)
	if path == "" {
		return nil
	}
	entries, err := readHostTokens(path)
	if err != nil {
		return err
	}
	entries = upsertHostToken(entries, hostTokenEntry{ID: id, Name: name, Token: token})
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal host tokens: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create host tokens dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write host tokens file: %w", err)
	}
	return nil
}

func readHostTokens(path string) ([]hostTokenEntry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read host tokens file: %w", err)
	}
	var entries []hostTokenEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse host tokens file: %w", err)
	}
	return entries, nil
}

func upsertHostToken(entries []hostTokenEntry, next hostTokenEntry) []hostTokenEntry {
	for i, e := range entries {
		if e.ID == next.ID {
			entries[i] = next
			return entries
		}
	}
	return append(entries, next)
}
