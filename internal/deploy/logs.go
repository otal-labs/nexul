package deploy

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// DefaultLogTail is how many lines a container log read starts from when the caller names none.
const DefaultLogTail = containerlog.DefaultTail

const (
	logMask       = "••••"
	logMaskMinLen = 6
)

// ContainerLogLine is one line a service's container printed.
type ContainerLogLine = containerlog.Line

// LogFeed is one open container log stream: Next returns the lines since the last call, io.EOF once it ended.
type LogFeed = containerlog.Feed

// LogSource is the runner seam that streams a container's logs from its machine (ADR 0091).
type LogSource interface {
	OpenLogs(ctx context.Context, machine, container string, tail int, follow bool) (LogFeed, error)
}

// SetLogSource wires the runner's log streams; unset, reading logs answers that no runner can serve them.
func (s *Service) SetLogSource(l LogSource) { s.logs = l }

// ServiceLogs opens a service's container logs with the stack's env values masked; Close stops the stream (ADR 0091).
func (s *Service) ServiceLogs(ctx context.Context, stackID, service string, tail int, follow bool) (LogFeed, error) {
	if tail < 0 {
		return nil, fmt.Errorf("%w: tail must be zero or more lines", apperrs.ErrInvalid)
	}
	stack, err := s.loadStack(ctx, stackID, permissions.StacksLogs)
	if err != nil {
		return nil, fmt.Errorf("logs of %s: %w", service, err)
	}
	container, err := s.containerOf(ctx, stack, service)
	if err != nil {
		return nil, err
	}
	if s.logs == nil {
		return nil, fmt.Errorf("%w: no runner can serve container logs", apperrs.ErrRetryable)
	}
	feed, err := s.logs.OpenLogs(ctx, stack.Machine, container, tail, follow)
	if err != nil {
		return nil, fmt.Errorf("logs of %s: %w", service, err)
	}
	return maskedFeed{LogFeed: feed, mask: envMask(stack.Env)}, nil
}

// ServiceLogSnapshot reads the last tail lines of a service's container without following them.
func (s *Service) ServiceLogSnapshot(ctx context.Context, stackID, service string, tail int) ([]ContainerLogLine, error) {
	return containerlog.Snapshot(ctx, s.serviceLogOpener(stackID, service), tail, service)
}

func (s *Service) serviceLogOpener(stackID, service string) containerlog.Open {
	return func(ctx context.Context, tail int, follow bool) (containerlog.Feed, error) {
		return s.ServiceLogs(ctx, stackID, service, tail, follow)
	}
}

// containerOf resolves a stack's service to the Docker container its last deploy started.
func (s *Service) containerOf(ctx context.Context, stack *Stack, service string) (string, error) {
	containers, err := s.services.ListByStack(ctx, stack.ID)
	if err != nil {
		return "", fmt.Errorf("logs of %s: %w", service, err)
	}
	i := slices.IndexFunc(containers, func(c *Container) bool { return c.Name == service })
	if i < 0 {
		return "", fmt.Errorf("%w: stack %s has no service %q", apperrs.ErrNotFound, stack.Name, service)
	}
	if containers[i].ContainerName == "" {
		return "", fmt.Errorf("%w: service %q has not started a container yet", apperrs.ErrConflict, service)
	}
	return containers[i].ContainerName, nil
}

// envMask masks env values of logMaskMinLen or more characters, longest first so one holding another goes whole.
func envMask(env map[string]string) *strings.Replacer {
	var values []string
	for _, v := range env {
		if utf8.RuneCountInString(v) >= logMaskMinLen {
			values = append(values, v)
		}
	}
	slices.SortFunc(values, func(a, b string) int { return len(b) - len(a) })
	pairs := make([]string, 0, 2*len(values))
	for _, v := range values {
		pairs = append(pairs, v, logMask)
	}
	return strings.NewReplacer(pairs...)
}

type maskedFeed struct {
	LogFeed
	mask *strings.Replacer
}

func (m maskedFeed) Next(ctx context.Context) ([]ContainerLogLine, error) {
	lines, err := m.LogFeed.Next(ctx)
	for i := range lines {
		lines[i].Line = m.mask.Replace(lines[i].Line)
	}
	return lines, err
}
