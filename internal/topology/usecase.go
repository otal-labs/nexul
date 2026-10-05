package topology

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Service is the topology use-case layer (ADR 0019); mutations enqueue topology.updated via the outbox.
type Service struct {
	repo  Repo
	gate  Gate
	scope Scope
}

// Gate is the permission check the canvas passes through: a workspace's canvas in that workspace, the service
// registry the deploy consumers keep in any workspace the caller belongs to (the access domain, ADR 0042).
type Gate interface {
	Require(ctx context.Context, workspaceID string, action permissions.Action) error
	RequireAnywhere(ctx context.Context, action permissions.Action) error
}

// Scope names the service nodes a workspace's canvas shows: its projects' containers and the gateways routing to
// them (ADR 0125). Each id is a container id or, for nodes saved before containers existed, a stack id.
type Scope interface {
	Services(ctx context.Context, workspaceID string) (map[string]bool, error)
}

// SetGate wires the permission check; unset, only the server's own calls pass.
func (s *Service) SetGate(g Gate) { s.gate = g }

// SetScope wires which service nodes each workspace shows; unset, every workspace shows every service node.
func (s *Service) SetScope(sc Scope) { s.scope = sc }

func (s *Service) require(ctx context.Context, workspaceID string, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.Require(ctx, workspaceID, action)
}

func (s *Service) requireRegistry(ctx context.Context, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.RequireAnywhere(ctx, action)
}

// NewService wires the topology use-cases over the given repo.
func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

// Get returns a workspace's canvas: its service nodes from the registry, placed where the workspace left them,
// with the nodes and edges its people drew.
func (s *Service) Get(ctx context.Context, workspaceID string) (*Canvas, error) {
	if err := s.require(ctx, workspaceID, permissions.TopologyRead); err != nil {
		return nil, err
	}
	return s.view(ctx, workspaceID)
}

// Update persists a workspace's full canvas; service nodes are auto-managed, so only their positions are taken.
func (s *Service) Update(ctx context.Context, workspaceID string, c *Canvas) (*Canvas, error) {
	if err := s.require(ctx, workspaceID, permissions.TopologyWrite); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	stored, err := s.view(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	merged := *c
	merged.Nodes = make([]Node, 0, len(c.Nodes)+len(stored.Nodes))
	incoming := make(map[string]Node, len(c.Nodes))
	for _, n := range c.Nodes {
		incoming[serviceKey(n)] = n
		if n.Type != NodeService {
			merged.Nodes = append(merged.Nodes, n)
		}
	}
	// Only the service nodes the workspace shows survive; identity, name, and status come from the registry.
	for _, sn := range stored.Nodes {
		if sn.Type != NodeService {
			continue
		}
		if n, ok := incoming[serviceKey(sn)]; ok {
			sn.Position = n.Position
		}
		merged.Nodes = append(merged.Nodes, sn)
	}
	merged.Edges = edgesBetween(merged.Nodes, c.Edges)
	if err := s.save(ctx, workspaceID, &merged); err != nil {
		return nil, err
	}
	return &merged, nil
}

// AddNode inserts a drawn node on a workspace's canvas; service nodes come and go with stacks instead.
func (s *Service) AddNode(ctx context.Context, workspaceID string, n Node) (*Canvas, error) {
	if err := s.require(ctx, workspaceID, permissions.TopologyWrite); err != nil {
		return nil, err
	}
	if n.Type == NodeService {
		return nil, fmt.Errorf("%w: node %q: service nodes are managed from stacks; add a network or external node", apperrs.ErrInvalid, n.ID)
	}
	if err := n.Validate(); err != nil {
		return nil, fmt.Errorf("add node: %w", err)
	}
	c, err := s.view(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if err := appendNode(c, n); err != nil {
		return nil, err
	}
	if err := s.save(ctx, workspaceID, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RemoveNode deletes a drawn node and its edges from a workspace's canvas; service nodes leave with their stack.
func (s *Service) RemoveNode(ctx context.Context, workspaceID string, nodeID string) (*Canvas, error) {
	if err := s.require(ctx, workspaceID, permissions.TopologyDelete); err != nil {
		return nil, err
	}
	if nodeID == "" {
		return nil, fmt.Errorf("%w: node id is required", apperrs.ErrInvalid)
	}
	c, err := s.view(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, n := range c.Nodes {
		if n.ID == nodeID && n.Type == NodeService {
			return nil, fmt.Errorf("%w: service node %q is auto-managed and cannot be removed on the canvas", apperrs.ErrInvalid, nodeID)
		}
	}
	if err := dropNode(c, nodeID); err != nil {
		return nil, err
	}
	if err := s.save(ctx, workspaceID, c); err != nil {
		return nil, err
	}
	return c, nil
}

// AddEdge inserts a relation between two nodes on a workspace's canvas, validating the canvas first.
func (s *Service) AddEdge(ctx context.Context, workspaceID string, e Edge) (*Canvas, error) {
	if err := s.require(ctx, workspaceID, permissions.TopologyWrite); err != nil {
		return nil, err
	}
	if err := e.Validate(); err != nil {
		return nil, fmt.Errorf("add edge: %w", err)
	}
	c, err := s.view(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, existing := range c.Edges {
		if existing.ID == e.ID {
			return nil, fmt.Errorf("add edge: %w: edge %q already exists", apperrs.ErrConflict, e.ID)
		}
	}
	c.Edges = append(c.Edges, e)
	if err := s.save(ctx, workspaceID, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RemoveEdge deletes a single relation by id from a workspace's canvas.
func (s *Service) RemoveEdge(ctx context.Context, workspaceID string, edgeID string) (*Canvas, error) {
	if err := s.require(ctx, workspaceID, permissions.TopologyDelete); err != nil {
		return nil, err
	}
	if edgeID == "" {
		return nil, fmt.Errorf("%w: edge id is required", apperrs.ErrInvalid)
	}
	c, err := s.view(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	kept := make([]Edge, 0, len(c.Edges))
	for _, e := range c.Edges {
		if e.ID != edgeID {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(c.Edges) {
		return nil, fmt.Errorf("%w: edge %q not found", apperrs.ErrNotFound, edgeID)
	}
	c.Edges = kept
	if err := s.save(ctx, workspaceID, c); err != nil {
		return nil, err
	}
	return c, nil
}

// AddServiceNode anchors a service node in the registry; one already there is a conflict.
func (s *Service) AddServiceNode(ctx context.Context, n Node) (*Canvas, error) {
	if err := s.requireRegistry(ctx, permissions.TopologyWrite); err != nil {
		return nil, err
	}
	n.Type = NodeService
	n.Data.ServiceID = n.ID
	if err := n.Validate(); err != nil {
		return nil, fmt.Errorf("add node: %w", err)
	}
	c, err := s.load(ctx, registryKey)
	if err != nil {
		return nil, err
	}
	if err := appendNode(c, n); err != nil {
		return nil, err
	}
	if err := s.save(ctx, registryKey, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RemoveServiceNode removes the registry node anchored to serviceID; a missing node is a no-op.
func (s *Service) RemoveServiceNode(ctx context.Context, serviceID string) (*Canvas, error) {
	if err := s.requireRegistry(ctx, permissions.TopologyDelete); err != nil {
		return nil, err
	}
	if serviceID == "" {
		return nil, fmt.Errorf("%w: service id is required", apperrs.ErrInvalid)
	}
	c, err := s.load(ctx, registryKey)
	if err != nil {
		return nil, err
	}
	n := serviceNode(c, serviceID)
	if n == nil {
		return c, nil
	}
	if err := dropNode(c, n.ID); err != nil {
		return nil, err
	}
	if err := s.save(ctx, registryKey, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RenameServiceNode updates the anchored service node's display name; a missing node is a no-op.
func (s *Service) RenameServiceNode(ctx context.Context, serviceID string, name string) (*Canvas, error) {
	if err := s.requireRegistry(ctx, permissions.TopologyWrite); err != nil {
		return nil, err
	}
	if serviceID == "" {
		return nil, fmt.Errorf("%w: service id is required", apperrs.ErrInvalid)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: service name is required", apperrs.ErrInvalid)
	}
	c, err := s.load(ctx, registryKey)
	if err != nil {
		return nil, err
	}
	if n := serviceNode(c, serviceID); n != nil {
		n.Data.Name = name
	}
	if err := s.save(ctx, registryKey, c); err != nil {
		return nil, err
	}
	return c, nil
}

// SetServiceNodeStatus updates the anchored node's live status badge and, when reported, its address;
// an empty address keeps whatever was set before, so a failed redeploy doesn't wipe the last known one.
func (s *Service) SetServiceNodeStatus(ctx context.Context, serviceID string, status ServiceStatus, address string) (*Canvas, error) {
	if err := s.requireRegistry(ctx, permissions.TopologyWrite); err != nil {
		return nil, err
	}
	if serviceID == "" {
		return nil, fmt.Errorf("%w: service id is required", apperrs.ErrInvalid)
	}
	if !status.Valid() {
		return nil, fmt.Errorf("%w: status %q is invalid", apperrs.ErrInvalid, status)
	}
	c, err := s.load(ctx, registryKey)
	if err != nil {
		return nil, err
	}
	if n := serviceNode(c, serviceID); n != nil {
		n.Data.Status = status
		if address != "" {
			n.Data.Address = address
		}
	}
	if err := s.save(ctx, registryKey, c); err != nil {
		return nil, err
	}
	return c, nil
}

// view composes a workspace's canvas: the registry's service nodes it shows at the positions it saved (unplaced at
// the origin, which the web lays out), its drawn nodes, and the edges whose ends are both still on it.
func (s *Service) view(ctx context.Context, workspaceID string) (*Canvas, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace is required", apperrs.ErrInvalid)
	}
	registry, err := s.load(ctx, registryKey)
	if err != nil {
		return nil, err
	}
	saved, err := s.load(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	shown, err := s.shown(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	placed := make(map[string]Position, len(saved.Nodes))
	out := &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{}, Viewport: saved.Viewport}
	for _, n := range saved.Nodes {
		if n.Type == NodeService {
			placed[serviceKey(n)] = n.Position
			continue
		}
		out.Nodes = append(out.Nodes, n)
	}
	for _, n := range registry.Nodes {
		if n.Type != NodeService || !shown(n) {
			continue
		}
		n.Position = placed[serviceKey(n)]
		out.Nodes = append(out.Nodes, n)
	}
	out.Edges = edgesBetween(out.Nodes, saved.Edges)
	return out, nil
}

// shown reports whether a registry node belongs on the workspace's canvas.
func (s *Service) shown(ctx context.Context, workspaceID string) (func(Node) bool, error) {
	if s.scope == nil {
		return func(Node) bool { return true }, nil
	}
	ids, err := s.scope.Services(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("scope topology %s: %w", workspaceID, err)
	}
	return func(n Node) bool { return ids[n.ID] || ids[n.Data.ServiceID] }, nil
}

// load returns a stored canvas, empty if missing, migrated in memory if its schema is below current.
func (s *Service) load(ctx context.Context, key string) (*Canvas, error) {
	c, err := s.repo.Get(ctx, key)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{}, Edges: []Edge{}}, nil
		}
		return nil, fmt.Errorf("load topology %s: %w", key, err)
	}
	c, _ = Migrate(c)
	return c, nil
}

// save persists the canvas and enqueues topology.updated via the outbox in the same transaction.
func (s *Service) save(ctx context.Context, key string, c *Canvas) error {
	if err := c.Validate(); err != nil {
		return err
	}
	payload := UpdatedEvent{Environment: key, Canvas: *c}
	if key != registryKey {
		payload.WorkspaceID = key
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: payload}
	if err := s.repo.Save(ctx, key, c, evt); err != nil {
		return fmt.Errorf("save topology %s: %w", key, err)
	}
	return nil
}

// serviceKey identifies a node across the registry and a workspace's saved positions.
func serviceKey(n Node) string {
	if n.Type == NodeService && n.Data.ServiceID != "" {
		return n.Data.ServiceID
	}
	return n.ID
}

func serviceNode(c *Canvas, serviceID string) *Node {
	for i := range c.Nodes {
		n := &c.Nodes[i]
		if n.Type == NodeService && (n.Data.ServiceID == serviceID || n.ID == serviceID) {
			return n
		}
	}
	return nil
}

func appendNode(c *Canvas, n Node) error {
	for _, existing := range c.Nodes {
		if existing.ID == n.ID {
			return fmt.Errorf("add node: %w: node %q already exists", apperrs.ErrConflict, n.ID)
		}
	}
	c.Nodes = append(c.Nodes, n)
	return nil
}

// dropNode removes a node and every edge referencing it, so edges always reference existing nodes.
func dropNode(c *Canvas, nodeID string) error {
	kept := make([]Node, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		if n.ID != nodeID {
			kept = append(kept, n)
		}
	}
	if len(kept) == len(c.Nodes) {
		return fmt.Errorf("%w: node %q not found", apperrs.ErrNotFound, nodeID)
	}
	c.Nodes = kept
	c.Edges = edgesBetween(kept, c.Edges)
	return nil
}

// edgesBetween keeps the edges whose ends are both among nodes.
func edgesBetween(nodes []Node, edges []Edge) []Edge {
	present := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		present[n.ID] = true
	}
	out := make([]Edge, 0, len(edges))
	for _, e := range edges {
		if present[e.Source] && present[e.Target] {
			out = append(out, e)
		}
	}
	return out
}
