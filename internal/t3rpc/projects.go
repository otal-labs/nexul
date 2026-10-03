package t3rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// listProjectsTimeout bounds the wait for the shell snapshot; a slow answer means a broken server, not a big list.
const listProjectsTimeout = 15 * time.Second

// ListProjects reads the registry off orchestration.subscribeShell's first snapshot item, since no list RPC exists.
func (c *Conn) ListProjects(ctx context.Context) ([]harness.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, listProjectsTimeout)
	defer cancel()

	stream, err := c.Stream(ctx, "orchestration.subscribeShell", nil)
	if err != nil {
		return nil, fmt.Errorf("subscribe shell: %w", err)
	}
	defer stream.Close()

	for {
		values, err := stream.Next(ctx)
		if errors.Is(err, ErrConnectionLost) {
			return nil, err
		}
		if ctx.Err() != nil {
			return nil, apperrs.Retryable(fmt.Errorf("list projects: %w", ctx.Err()))
		}
		if err != nil {
			return nil, apperrs.Retryable(fmt.Errorf("shell subscription ended before a snapshot arrived"))
		}
		for _, raw := range values {
			var item struct {
				Kind     string `json:"kind"`
				Snapshot struct {
					Projects []struct {
						ID            string  `json:"id"`
						Title         string  `json:"title"`
						WorkspaceRoot string  `json:"workspaceRoot"`
						DeletedAt     *string `json:"deletedAt"`
					} `json:"projects"`
				} `json:"snapshot"`
			}
			if err := json.Unmarshal(raw, &item); err != nil || item.Kind != "snapshot" {
				continue
			}
			projects := make([]harness.Project, 0, len(item.Snapshot.Projects))
			for _, p := range item.Snapshot.Projects {
				if p.DeletedAt != nil || p.ID == "" {
					continue
				}
				projects = append(projects, harness.Project{ID: p.ID, Title: p.Title, Path: p.WorkspaceRoot})
			}
			return projects, nil
		}
	}
}
