package topology

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestServiceStatus_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status ServiceStatus
		want   bool
	}{
		{"healthy", ServiceHealthy, true},
		{"running", ServiceRunning, true},
		{"stopped", ServiceStopped, true},
		{"failed", ServiceFailed, true},
		{"empty", "", false},
		{"bogus", "bogus", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Valid())
		})
	}
}

func TestRelationKind_Valid(t *testing.T) {
	tests := []struct {
		name string
		kind RelationKind
		want bool
	}{
		{"depends_on", KindDependsOn, true},
		{"connects_to", KindConnectsTo, true},
		{"mounts", KindMounts, true},
		{"empty", "", false},
		{"bogus", "bogus", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.kind.Valid())
		})
	}
}

func TestCanvas_Validate(t *testing.T) {
	t.Run("nil canvas", func(t *testing.T) {
		err := (*Canvas)(nil).Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	valid := func() *Canvas {
		return &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNode(), {ID: "svc-db", Type: NodeService, Position: Position{X: 1, Y: 1}, Data: NodeData{ServiceID: "svc-db", Name: "db", Runtime: "postgres", Status: ServiceHealthy}}},
			Edges:         []Edge{validEdge()},
		}
	}
	tests := []struct {
		name    string
		mutate  func(*Canvas)
		wantErr bool
	}{
		{"unsupported schema_version", func(c *Canvas) { c.SchemaVersion = 3 }, true},
		{"duplicate node id", func(c *Canvas) { c.Nodes[1].ID = c.Nodes[0].ID }, true},
		{"edge source is not a node", func(c *Canvas) { c.Edges[0].Source = "ghost" }, true},
		{"edge target is not a node", func(c *Canvas) { c.Edges[0].Target = "ghost" }, true},
		{"self-referencing edge", func(c *Canvas) { c.Edges[0].Target = c.Edges[0].Source }, true},
		{"duplicate edge id", func(c *Canvas) {
			c.Edges = append(c.Edges, Edge{ID: c.Edges[0].ID, Source: "svc-api", Target: "svc-db", Type: "relation", Data: EdgeData{Kind: KindConnectsTo}})
		}, true},
		{"valid canvas", func(*Canvas) {}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := valid()
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNode_Validate(t *testing.T) {
	t.Run("nil node", func(t *testing.T) {
		err := (*Node)(nil).Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("service node", func(t *testing.T) {
		tests := []struct {
			name    string
			mutate  func(*Node)
			wantErr bool
		}{
			{"empty id", func(n *Node) { n.ID = "" }, true},
			{"unsupported type", func(n *Node) { n.Type = "database" }, true},
			{"nan position", func(n *Node) { n.Position.X = math.NaN() }, true},
			{"inf position", func(n *Node) { n.Position.Y = math.Inf(1) }, true},
			{"missing service_id", func(n *Node) { n.Data.ServiceID = "" }, true},
			{"empty name", func(n *Node) { n.Data.Name = "" }, true},
			{"invalid status", func(n *Node) { n.Data.Status = ServiceStatus("bogus") }, true},
			{"valid node", func(*Node) {}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				n := validNode()
				tt.mutate(&n)
				err := n.Validate()
				if tt.wantErr {
					require.Error(t, err)
					assert.True(t, errors.Is(err, apperrs.ErrInvalid))
					return
				}
				require.NoError(t, err)
			})
		}
	})
	t.Run("network node requires name", func(t *testing.T) {
		n := validNetworkNode()
		n.Data.Name = ""
		err := n.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("network node validates", func(t *testing.T) {
		nw := validNetworkNode()
		require.NoError(t, nw.Validate())
		_ = 0
	})
	t.Run("external node requires label", func(t *testing.T) {
		n := validExternalNode()
		n.Data.Label = ""
		err := n.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("external node rejects bad label", func(t *testing.T) {
		n := validExternalNode()
		n.Data.Label = ExternalLabel("bogus")
		err := n.Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("external node validates", func(t *testing.T) {
		ex := validExternalNode()
		require.NoError(t, ex.Validate())
	})
}

func TestEdge_Validate(t *testing.T) {
	t.Run("nil edge", func(t *testing.T) {
		err := (*Edge)(nil).Validate()
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	valid := func() Edge { return validEdge() }
	tests := []struct {
		name    string
		mutate  func(*Edge)
		wantErr bool
	}{
		{"empty id", func(e *Edge) { e.ID = "" }, true},
		{"unsupported type", func(e *Edge) { e.Type = "solid" }, true},
		{"empty source", func(e *Edge) { e.Source = "" }, true},
		{"empty target", func(e *Edge) { e.Target = "" }, true},
		{"self edge", func(e *Edge) { e.Target = e.Source }, true},
		{"invalid kind", func(e *Edge) { e.Data.Kind = RelationKind("bogus") }, true},
		{"valid edge", func(*Edge) {}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := valid()
			tt.mutate(&e)
			err := e.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
				return
			}
			require.NoError(t, err)
		})
	}
}
