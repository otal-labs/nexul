package t3client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// listProjectsTimeout bounds the wait for the shell snapshot; a slow answer means a broken server, not a big list.
const listProjectsTimeout = 15 * time.Second

// ListProjects reads the registry off orchestration.subscribeShell's first snapshot item, since no list RPC exists.
func (c *Client) ListProjects(ctx context.Context) ([]harness.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, listProjectsTimeout)
	defer cancel()

	id, ch := c.register()
	defer func() {
		c.unregister(id)
		_ = c.send(c.ctx, interruptEnvelope{Tag: "Interrupt", RequestID: id})
	}()
	env := requestEnvelope{Tag: "Request", ID: id, RPCTag: "orchestration.subscribeShell", Payload: struct{}{}, Headers: [][]string{}}
	if err := c.send(ctx, env); err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("subscribe shell: %w", err))
	}

	for {
		select {
		case env := <-ch:
			switch env.Tag {
			case "Chunk":
				for _, raw := range env.Values {
					var item struct {
						Kind     string `json:"kind"`
						Snapshot struct {
							Projects []struct {
								ID        string  `json:"id"`
								Title     string  `json:"title"`
								DeletedAt *string `json:"deletedAt"`
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
						projects = append(projects, harness.Project{ID: p.ID, Title: p.Title})
					}
					return projects, nil
				}
				c.ack(id)
			case "Exit", "Defect":
				return nil, apperrs.Retryable(fmt.Errorf("shell subscription ended before a snapshot arrived"))
			}
		case <-ctx.Done():
			return nil, apperrs.Retryable(fmt.Errorf("list projects: %w", ctx.Err()))
		case <-c.done:
			return nil, apperrs.Retryable(fmt.Errorf("T3 connection lost: %v", c.err))
		}
	}
}
