package repository

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
)

// scanEnvKeys fetches every .env.example/.env.sample file found and merges their keys, first-seen order, deduped.
func scanEnvKeys(ctx context.Context, s Scanner, owner, name, ref string, paths []string) ([]string, error) {
	seen := map[string]bool{}
	var keys []string
	for _, p := range paths {
		b, err := s.GetFile(ctx, owner, name, ref, p)
		if err != nil {
			return nil, fmt.Errorf("get env file %s: %w", p, err)
		}
		for _, k := range parseEnvFileKeys(b) {
			if seen[k] {
				continue
			}
			seen[k] = true
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// parseEnvFileKeys extracts the left-hand key from each KEY=VALUE line, skipping blank lines and comments.
func parseEnvFileKeys(b []byte) []string {
	var keys []string
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		if key := strings.TrimSpace(line[:i]); key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}
