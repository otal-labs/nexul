package pairing

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

// mcpCommandsListed is how many recent commands computer_list returns with commands.
const mcpCommandsListed = 10

type computerCommandsIn struct {
	JobID string `json:"job_id,omitempty" jsonschema:"One command's job_id, to read it with its output; omit for the newest 10 without output."`
}

// commandsFor reads one command with its output, or a computer's newest commands without it.
func commandsFor(ctx context.Context, s *Service, userID string, c Computer, in computerCommandsIn) ([]shelljob.Job, error) {
	if in.JobID == "" {
		jobs, _, err := s.ListShellJobs(ctx, userID, c.ID, paging.Window{Limit: mcpCommandsListed})
		return jobs, err
	}
	job, err := s.GetShellJob(ctx, userID, in.JobID)
	if err != nil {
		return nil, err
	}
	if job.ComputerID != c.ID {
		return nil, fmt.Errorf("%w: command %s did not run on %s", apperrs.ErrNotFound, in.JobID, c.Name)
	}
	job.TailForAgent()
	return []shelljob.Job{*job}, nil
}
