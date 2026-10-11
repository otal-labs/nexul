package pairing

import (
	"context"
	"fmt"
	"regexp"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Containers reaches a computer's Docker through its runner, with the discover and logs_request a machine's answers (ADR 0091).
type Containers interface {
	// ComputerContainers lists every container on computerID, running or not.
	ComputerContainers(ctx context.Context, computerID string) ([]Container, error)
	// OpenComputerLogs streams one container's last tail lines, and new ones while follow; the caller closes the feed.
	OpenComputerLogs(ctx context.Context, computerID, container string, tail int, follow bool) (containerlog.Feed, error)
}

// Container is one Docker container on a person's computer. It is not a Nexul service: nothing imports or deploys it.
type Container struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
}

// containerName is Docker's own name rule; a leading dash would read as a flag to docker logs.
var containerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,254}$`)

// ListContainers lists the containers on one of the caller's computers whose runner reported Docker.
func (s *Service) ListContainers(ctx context.Context, userID, computerID string) ([]Container, error) {
	c, err := s.dockerComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	containers, err := s.containers.ComputerContainers(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("containers on %s: %w", c.Name, err)
	}
	return containers, nil
}

// ContainerLogs opens one container's logs on one of the caller's computers, unmasked: the output is their own.
func (s *Service) ContainerLogs(ctx context.Context, userID, computerID, container string, tail int, follow bool) (containerlog.Feed, error) {
	if tail < 0 {
		return nil, fmt.Errorf("%w: tail must be zero or more lines", apperrs.ErrInvalid)
	}
	c, err := s.dockerComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if !containerName.MatchString(container) {
		return nil, fmt.Errorf("%w: %q is not a container name", apperrs.ErrInvalid, container)
	}
	return s.containers.OpenComputerLogs(ctx, c.ID, container, tail, follow)
}

// ContainerLogSnapshot reads the last tail lines of one container on one of the caller's computers.
func (s *Service) ContainerLogSnapshot(ctx context.Context, userID, computerID, container string, tail int) ([]containerlog.Line, error) {
	return containerlog.Snapshot(ctx, s.containerLogOpener(userID, computerID, container), tail, container)
}

func (s *Service) containerLogOpener(userID, computerID, container string) containerlog.Open {
	return func(ctx context.Context, tail int, follow bool) (containerlog.Feed, error) {
		return s.ContainerLogs(ctx, userID, computerID, container, tail, follow)
	}
}

// dockerComputer is one of the caller's computers whose runner last reported a Docker daemon; anyone else's is not found.
func (s *Service) dockerComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if c.Facts == nil || c.Facts.Docker == nil || !c.Facts.Docker.Available {
		return nil, fmt.Errorf("%w: %s has no Docker its Nexul app can reach", apperrs.ErrConflict, c.Name)
	}
	if s.containers == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: computer containers are not wired", apperrs.ErrFatal))
	}
	return c, nil
}
