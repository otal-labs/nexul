package runner

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logstore"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// LogReader is the slice of the log store the instance's own logs are read through (ADR 0155).
type LogReader interface {
	Read(ctx context.Context, q logstore.Query) (logstore.Page, error)
}

// WithLogs attaches the reader of the instance's logs; unset, a read is refused as an instance that keeps none.
func (s *Service) WithLogs(r LogReader) *Service {
	s.logs = r
	return s
}

// InstanceLogs reads the instance's own logs for logs:read, newest first.
func (s *Service) InstanceLogs(ctx context.Context, p logstore.Params) (logstore.Page, error) {
	if err := s.require(ctx, permissions.LogsRead); err != nil {
		return logstore.Page{}, err
	}
	q, err := p.Query(s.now())
	if err != nil {
		return logstore.Page{}, err
	}
	if s.logs == nil {
		return logstore.Page{}, fmt.Errorf("%w: this instance keeps no logs", apperrs.ErrInvalid)
	}
	return s.logs.Read(ctx, q)
}
