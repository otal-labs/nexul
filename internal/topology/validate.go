package topology

import (
	"fmt"
	"math"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// CurrentSchemaVersion is the only canvas schema the backend accepts; bump it with a migration path when the schema evolves.
const CurrentSchemaVersion = 2

// DefaultEnvironment is the canonical environment key for callers (MCP tools, resources) that do not pass one.
const DefaultEnvironment = "default"

// Valid reports whether s is one of the serialized status strings.
func (s ServiceStatus) Valid() bool {
	switch s {
	case ServiceHealthy, ServiceRunning, ServiceStopped, ServiceFailed:
		return true
	default:
		return false
	}
}

// Valid reports whether k is one of the serialized relation kinds.
func (k RelationKind) Valid() bool {
	switch k {
	case KindDependsOn, KindConnectsTo, KindMounts:
		return true
	default:
		return false
	}
}

// Valid reports whether t is one of the node kinds.
func (t NodeType) Valid() bool {
	switch t {
	case NodeService, NodeNetwork, NodeExternal:
		return true
	default:
		return false
	}
}

// Valid reports whether l is one of the external node labels.
func (l ExternalLabel) Valid() bool {
	switch l {
	case LabelDomain, LabelTunnel, LabelProxy, LabelDatabase, LabelAPI, LabelOther:
		return true
	default:
		return false
	}
}

// Validate enforces the canvas contract: a supported schema_version, and every edge referencing existing nodes.
func (c *Canvas) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: canvas is nil", apperrs.ErrInvalid)
	}
	if c.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: unsupported schema_version %d (want %d)", apperrs.ErrInvalid, c.SchemaVersion, CurrentSchemaVersion)
	}
	nodeIDs := make(map[string]struct{}, len(c.Nodes))
	for _, n := range c.Nodes {
		if err := n.Validate(); err != nil {
			return err
		}
		if _, dup := nodeIDs[n.ID]; dup {
			return fmt.Errorf("%w: duplicate node id %q", apperrs.ErrInvalid, n.ID)
		}
		nodeIDs[n.ID] = struct{}{}
	}
	edgeIDs := make(map[string]struct{}, len(c.Edges))
	for _, e := range c.Edges {
		if err := e.Validate(); err != nil {
			return err
		}
		if _, dup := edgeIDs[e.ID]; dup {
			return fmt.Errorf("%w: duplicate edge id %q", apperrs.ErrInvalid, e.ID)
		}
		if _, ok := nodeIDs[e.Source]; !ok {
			return fmt.Errorf("%w: edge %q source %q is not a node", apperrs.ErrInvalid, e.ID, e.Source)
		}
		if _, ok := nodeIDs[e.Target]; !ok {
			return fmt.Errorf("%w: edge %q target %q is not a node", apperrs.ErrInvalid, e.ID, e.Target)
		}
		edgeIDs[e.ID] = struct{}{}
	}
	return nil
}

// Validate enforces the per-node part of the canvas contract, per node kind.
func (n *Node) Validate() error {
	if n == nil {
		return fmt.Errorf("%w: node is nil", apperrs.ErrInvalid)
	}
	if n.ID == "" {
		return fmt.Errorf("%w: node id is required", apperrs.ErrInvalid)
	}
	if !n.Type.Valid() {
		return fmt.Errorf("%w: node type %q is not supported (want service, network, or external)", apperrs.ErrInvalid, n.Type)
	}
	if math.IsNaN(n.Position.X) || math.IsInf(n.Position.X, 0) || math.IsNaN(n.Position.Y) || math.IsInf(n.Position.Y, 0) {
		return fmt.Errorf("%w: node %q position must be finite", apperrs.ErrInvalid, n.ID)
	}
	switch n.Type {
	case NodeService:
		return n.validateServiceData()
	case NodeNetwork:
		return n.validateNamedData()
	case NodeExternal:
		return n.validateExternalData()
	}
	return nil
}

func (n *Node) validateServiceData() error {
	if n.Data.ServiceID == "" {
		return fmt.Errorf("%w: node %q service_id is required", apperrs.ErrInvalid, n.ID)
	}
	if n.Data.Name == "" {
		return fmt.Errorf("%w: node %q name is required", apperrs.ErrInvalid, n.ID)
	}
	if !n.Data.Status.Valid() {
		return fmt.Errorf("%w: node %q status %q is invalid", apperrs.ErrInvalid, n.ID, n.Data.Status)
	}
	return nil
}

func (n *Node) validateNamedData() error {
	if n.Data.Name == "" {
		return fmt.Errorf("%w: node %q name is required", apperrs.ErrInvalid, n.ID)
	}
	return nil
}

func (n *Node) validateExternalData() error {
	if err := n.validateNamedData(); err != nil {
		return err
	}
	if !n.Data.Label.Valid() {
		return fmt.Errorf("%w: node %q label %q is invalid", apperrs.ErrInvalid, n.ID, n.Data.Label)
	}
	return nil
}

// Validate enforces the per-edge part of the canvas contract.
func (e *Edge) Validate() error {
	if e == nil {
		return fmt.Errorf("%w: edge is nil", apperrs.ErrInvalid)
	}
	if e.ID == "" {
		return fmt.Errorf("%w: edge id is required", apperrs.ErrInvalid)
	}
	if e.Type != "relation" {
		return fmt.Errorf("%w: edge type %q is not supported (want relation)", apperrs.ErrInvalid, e.Type)
	}
	if e.Source == "" || e.Target == "" {
		return fmt.Errorf("%w: edge %q source and target are required", apperrs.ErrInvalid, e.ID)
	}
	if e.Source == e.Target {
		return fmt.Errorf("%w: edge %q cannot connect a node to itself", apperrs.ErrInvalid, e.ID)
	}
	if !e.Data.Kind.Valid() {
		return fmt.Errorf("%w: edge %q kind %q is invalid", apperrs.ErrInvalid, e.ID, e.Data.Kind)
	}
	return nil
}
