package topology

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Service is the topology use-case layer (ADR 0019); mutations enqueue topology.updated via the outbox.
type Service struct {
	repo Repo
}

// NewService wires the topology use-cases over the given repo.
func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

// Get returns the canvas for an environment, empty if unsaved; stored canvases below schema are migrated on read.
func (s *Service) Get(ctx context.Context, environment string) (*Canvas, error) {
	if environment == "" {
		return nil, fmt.Errorf("%w: environment is required", apperrs.ErrInvalid)
	}
	c, err := s.repo.Get(ctx, environment)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{}, Edges: []Edge{}}, nil
		}
		return nil, fmt.Errorf("get topology %s: %w", environment, err)
	}
	c, migrated := Migrate(c)
	if migrated {
		if err := s.repo.Save(ctx, environment, c); err != nil {
			return nil, fmt.Errorf("migrate topology %s: %w", environment, err)
		}
	}
	return c, nil
}

// Update persists a full canvas; service nodes are auto-managed, so stored ones are merged back in.
func (s *Service) Update(ctx context.Context, environment string, c *Canvas) (*Canvas, error) {
	if environment == "" {
		return nil, fmt.Errorf("%w: environment is required", apperrs.ErrInvalid)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	stored, err := s.load(ctx, environment)
	if err != nil {
		return nil, err
	}
	merged := *c
	merged.Nodes = make([]Node, 0, len(c.Nodes)+len(stored.Nodes))
	incoming := make(map[string]Node, len(c.Nodes))
	for _, n := range c.Nodes {
		key := n.ID
		if n.Type == NodeService {
			key = n.Data.ServiceID
		}
		incoming[key] = n
	}
	for _, n := range c.Nodes {
		if n.Type != NodeService {
			merged.Nodes = append(merged.Nodes, n)
		}
	}
	// Only nodes already in storage survive; identity, name, and status come from the stored node.
	for _, sn := range stored.Nodes {
		if sn.Type != NodeService {
			continue
		}
		if n, ok := incoming[sn.Data.ServiceID]; ok {
			sn.Position = n.Position
		}
		merged.Nodes = append(merged.Nodes, sn)
	}
	if err := merged.Validate(); err != nil {
		return nil, err
	}
	if err := s.save(ctx, environment, &merged); err != nil {
		return nil, err
	}
	return &merged, nil
}

// AddNode inserts a node; a service node must anchor to an existing service definition by ID.
func (s *Service) AddNode(ctx context.Context, environment string, n Node) (*Canvas, error) {
	if n.Type == NodeService {
		n.Data.ServiceID = n.ID
	}
	if err := n.Validate(); err != nil {
		return nil, fmt.Errorf("add node: %w", err)
	}
	c, err := s.load(ctx, environment)
	if err != nil {
		return nil, err
	}
	for _, existing := range c.Nodes {
		if existing.ID == n.ID {
			return nil, fmt.Errorf("add node: %w: node %q already exists", apperrs.ErrConflict, n.ID)
		}
	}
	c.Nodes = append(c.Nodes, n)
	if err := s.save(ctx, environment, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RemoveNode deletes a non-service node and its edges; service nodes go through RemoveServiceNode instead.
func (s *Service) RemoveNode(ctx context.Context, environment string, nodeID string) (*Canvas, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("%w: node id is required", apperrs.ErrInvalid)
	}
	c, err := s.load(ctx, environment)
	if err != nil {
		return nil, err
	}
	for _, n := range c.Nodes {
		if n.ID == nodeID && n.Type == NodeService {
			return nil, fmt.Errorf("%w: service node %q is auto-managed and cannot be removed on the canvas", apperrs.ErrInvalid, nodeID)
		}
	}
	return s.removeNode(ctx, environment, c, nodeID)
}

// RemoveServiceNode removes the node anchored to serviceID and its edges; a missing node is a no-op.
func (s *Service) RemoveServiceNode(ctx context.Context, environment string, serviceID string) (*Canvas, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("%w: service id is required", apperrs.ErrInvalid)
	}
	c, err := s.load(ctx, environment)
	if err != nil {
		return nil, err
	}
	nodeID := ""
	for _, n := range c.Nodes {
		if n.Type == NodeService && (n.Data.ServiceID == serviceID || n.ID == serviceID) {
			nodeID = n.ID
			break
		}
	}
	if nodeID == "" {
		return c, nil
	}
	return s.removeNode(ctx, environment, c, nodeID)
}

// removeNode drops a node and every edge referencing it, so edges always reference existing nodes.
func (s *Service) removeNode(ctx context.Context, environment string, c *Canvas, nodeID string) (*Canvas, error) {
	found := false
	kept := make([]Node, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		if n.ID == nodeID {
			found = true
			continue
		}
		kept = append(kept, n)
	}
	if !found {
		return nil, fmt.Errorf("%w: node %q not found", apperrs.ErrNotFound, nodeID)
	}
	c.Nodes = kept
	edges := make([]Edge, 0, len(c.Edges))
	for _, e := range c.Edges {
		if e.Source == nodeID || e.Target == nodeID {
			continue
		}
		edges = append(edges, e)
	}
	c.Edges = edges
	if err := s.save(ctx, environment, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RenameServiceNode updates the anchored service node's display name; a missing node is a no-op.
func (s *Service) RenameServiceNode(ctx context.Context, environment string, serviceID string, name string) (*Canvas, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("%w: service id is required", apperrs.ErrInvalid)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: service name is required", apperrs.ErrInvalid)
	}
	c, err := s.load(ctx, environment)
	if err != nil {
		return nil, err
	}
	for i := range c.Nodes {
		n := &c.Nodes[i]
		if n.Type == NodeService && (n.Data.ServiceID == serviceID || n.ID == serviceID) {
			n.Data.Name = name
			break
		}
	}
	if err := s.save(ctx, environment, c); err != nil {
		return nil, err
	}
	return c, nil
}

// SetServiceNodeStatus updates the anchored node's live status badge and, when reported, its address;
// an empty address keeps whatever was set before, so a failed redeploy doesn't wipe the last known one.
func (s *Service) SetServiceNodeStatus(ctx context.Context, environment string, serviceID string, status ServiceStatus, address string) (*Canvas, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("%w: service id is required", apperrs.ErrInvalid)
	}
	if !status.Valid() {
		return nil, fmt.Errorf("%w: status %q is invalid", apperrs.ErrInvalid, status)
	}
	c, err := s.load(ctx, environment)
	if err != nil {
		return nil, err
	}
	for i := range c.Nodes {
		n := &c.Nodes[i]
		if n.Type == NodeService && (n.Data.ServiceID == serviceID || n.ID == serviceID) {
			n.Data.Status = status
			if address != "" {
				n.Data.Address = address
			}
			break
		}
	}
	if err := s.save(ctx, environment, c); err != nil {
		return nil, err
	}
	return c, nil
}

// AddEdge inserts a relation between two nodes, validating the canvas first.
func (s *Service) AddEdge(ctx context.Context, environment string, e Edge) (*Canvas, error) {
	if err := e.Validate(); err != nil {
		return nil, fmt.Errorf("add edge: %w", err)
	}
	c, err := s.load(ctx, environment)
	if err != nil {
		return nil, err
	}
	for _, existing := range c.Edges {
		if existing.ID == e.ID {
			return nil, fmt.Errorf("add edge: %w: edge %q already exists", apperrs.ErrConflict, e.ID)
		}
	}
	c.Edges = append(c.Edges, e)
	if err := s.save(ctx, environment, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RemoveEdge deletes a single relation by id.
func (s *Service) RemoveEdge(ctx context.Context, environment string, edgeID string) (*Canvas, error) {
	if edgeID == "" {
		return nil, fmt.Errorf("%w: edge id is required", apperrs.ErrInvalid)
	}
	c, err := s.load(ctx, environment)
	if err != nil {
		return nil, err
	}
	found := false
	kept := make([]Edge, 0, len(c.Edges))
	for _, e := range c.Edges {
		if e.ID == edgeID {
			found = true
			continue
		}
		kept = append(kept, e)
	}
	if !found {
		return nil, fmt.Errorf("%w: edge %q not found", apperrs.ErrNotFound, edgeID)
	}
	c.Edges = kept
	if err := s.save(ctx, environment, c); err != nil {
		return nil, err
	}
	return c, nil
}

// load returns the current canvas, empty if missing, migrated in memory if its schema is below current.
func (s *Service) load(ctx context.Context, environment string) (*Canvas, error) {
	c, err := s.repo.Get(ctx, environment)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{}, Edges: []Edge{}}, nil
		}
		return nil, fmt.Errorf("load topology %s: %w", environment, err)
	}
	c, _ = Migrate(c)
	return c, nil
}

// save persists the canvas and enqueues topology.updated via the outbox in the same transaction.
func (s *Service) save(ctx context.Context, environment string, c *Canvas) error {
	if err := c.Validate(); err != nil {
		return err
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{Environment: environment, Canvas: *c}}
	if err := s.repo.Save(ctx, environment, c, evt); err != nil {
		return fmt.Errorf("save topology %s: %w", environment, err)
	}
	return nil
}
