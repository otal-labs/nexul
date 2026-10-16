package pairing

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Containers reaches a computer's Docker and its runner's own log through the runner, with the discover and logs_request a machine's answers (ADR 0091).
type Containers interface {
	// ComputerContainers lists every container on computerID, running or not.
	ComputerContainers(ctx context.Context, computerID string) ([]Container, error)
	// OpenComputerLogs streams one container's last tail lines, and new ones while follow, hiding its own env values when mask; the caller closes the feed.
	OpenComputerLogs(ctx context.Context, computerID, container string, tail int, follow, mask bool) (containerlog.Feed, error)
	// OpenRunnerLogs streams the last tail lines of computerID's runner's own log, and new ones while follow; the caller closes the feed.
	OpenRunnerLogs(ctx context.Context, computerID string, tail int, follow bool) (containerlog.Feed, error)
}

// Container is one Docker container on a person's computer. It is not a Nexul service: nothing imports or deploys it.
type Container struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
}

// containerName is Docker's own name rule; a leading dash would read as a flag to docker logs.
var containerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,254}$`)

// ListContainers lists the containers on a computer whose runner reported Docker, for its owner or a person its share lets read logs.
func (s *Service) ListContainers(ctx context.Context, userID, computerID string) ([]Container, error) {
	c, _, err := s.dockerComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	containers, err := s.containers.ComputerContainers(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("containers on %s: %w", c.Name, err)
	}
	return containers, nil
}

// ContainerLogs opens one container's logs on a computer: unmasked for its owner, env values hidden for anyone else.
func (s *Service) ContainerLogs(ctx context.Context, userID, computerID, container string, tail int, follow bool) (containerlog.Feed, error) {
	if tail < 0 {
		return nil, fmt.Errorf("%w: tail must be zero or more lines", apperrs.ErrInvalid)
	}
	c, mask, err := s.dockerComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if !containerName.MatchString(container) {
		return nil, fmt.Errorf("%w: %q is not a container name", apperrs.ErrInvalid, container)
	}
	return s.containers.OpenComputerLogs(ctx, c.ID, container, tail, follow, mask)
}

// RunnerLogs opens a computer's runner's own log for its owner or a person its share lets read logs; it needs no Docker.
func (s *Service) RunnerLogs(ctx context.Context, userID, computerID string, tail int, follow bool) (containerlog.Feed, error) {
	if tail < 0 {
		return nil, fmt.Errorf("%w: tail must be zero or more lines", apperrs.ErrInvalid)
	}
	c, err := s.logsComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if s.containers == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: computer logs are not wired", apperrs.ErrFatal))
	}
	return s.containers.OpenRunnerLogs(ctx, c.ID, tail, follow)
}

// RunnerLogSnapshot reads the last tail lines of a computer's runner's own log without following them.
func (s *Service) RunnerLogSnapshot(ctx context.Context, userID, computerID string, tail int) ([]containerlog.Line, error) {
	return containerlog.Snapshot(ctx, s.runnerLogOpener(userID, computerID), tail, "the runner")
}

func (s *Service) runnerLogOpener(userID, computerID string) containerlog.Open {
	return func(ctx context.Context, tail int, follow bool) (containerlog.Feed, error) {
		return s.RunnerLogs(ctx, userID, computerID, tail, follow)
	}
}

// ContainerLogSnapshot reads the last tail lines of one container on a computer.
func (s *Service) ContainerLogSnapshot(ctx context.Context, userID, computerID, container string, tail int) ([]containerlog.Line, error) {
	return containerlog.Snapshot(ctx, s.containerLogOpener(userID, computerID, container), tail, container)
}

func (s *Service) containerLogOpener(userID, computerID, container string) containerlog.Open {
	return func(ctx context.Context, tail int, follow bool) (containerlog.Feed, error) {
		return s.ContainerLogs(ctx, userID, computerID, container, tail, follow)
	}
}

// logsComputer is a computer the caller owns or may read the logs of; any other is not found, so a refusal says nothing.
func (s *Service) logsComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err == nil || !errors.Is(err, apperrs.ErrNotFound) || s.computerAccess == nil {
		return c, err
	}
	notFound := fmt.Errorf("%w: computer %s", apperrs.ErrNotFound, computerID)
	ownerID, held, err := s.computerAccess.ComputerPermissions(ctx, userID, computerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, notFound
	}
	if err != nil {
		return nil, fmt.Errorf("read the share of computer %s: %w", computerID, err)
	}
	if !held.Has(permissions.ComputersLogs) {
		return nil, notFound
	}
	return s.repo.GetComputer(ctx, ownerID, computerID)
}

// dockerComputer is a logsComputer whose runner last reported Docker; mask is true when it is not the caller's own.
func (s *Service) dockerComputer(ctx context.Context, userID, computerID string) (c *Computer, mask bool, err error) {
	c, err = s.logsComputer(ctx, userID, computerID)
	if err != nil {
		return nil, false, err
	}
	if c.Facts == nil || c.Facts.Docker == nil || !c.Facts.Docker.Available {
		return nil, false, fmt.Errorf("%w: %s has no Docker its Nexul app can reach", apperrs.ErrConflict, c.Name)
	}
	if s.containers == nil {
		return nil, false, apperrs.Fatal(fmt.Errorf("%w: computer containers are not wired", apperrs.ErrFatal))
	}
	return c, c.UserID != userID, nil
}
