package topology

type RelationKind string

const (
	KindDependsOn  RelationKind = "depends_on"
	KindConnectsTo RelationKind = "connects_to"
	KindMounts     RelationKind = "mounts"
)

type ServiceStatus string

const (
	ServiceHealthy ServiceStatus = "healthy"
	ServiceRunning ServiceStatus = "running"
	ServiceStopped ServiceStatus = "stopped"
	ServiceFailed  ServiceStatus = "failed"
)

// NodeType selects the node kind on the canvas. Each kind has its own data contract enforced by Node.Validate.
type NodeType string

const (
	// NodeService anchors to a deploy service definition by ID and renders its name, live status, and details.
	NodeService NodeType = "service"
	// NodeNetwork is a freely placeable Docker network hub.
	NodeNetwork NodeType = "network"
	// NodeExternal is a free-form element for things Nexul doesn't manage: domains, tunnels, third-party APIs.
	NodeExternal NodeType = "external"
)

// ExternalLabel is the user-picked label of an external node.
type ExternalLabel string

const (
	LabelDomain   ExternalLabel = "domain"
	LabelTunnel   ExternalLabel = "tunnel"
	LabelProxy    ExternalLabel = "proxy"
	LabelDatabase ExternalLabel = "database"
	LabelAPI      ExternalLabel = "api"
	LabelOther    ExternalLabel = "other"
)

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NodeData carries a node's per-kind payload; only the fields for its Type are meaningful.
type NodeData struct {
	// service nodes: anchored to a deploy service definition by ID.
	ServiceID string        `json:"service_id,omitempty"`
	Name      string        `json:"name,omitempty"`
	Runtime   string        `json:"runtime,omitempty"`
	URL       string        `json:"url,omitempty"`
	Status    ServiceStatus `json:"status,omitempty"`
	Replicas  int           `json:"replicas,omitempty"`
	Volume    string        `json:"volume,omitempty"`
	// Address is the container's address on its docker network, the one fact stored on a node rather than derived (ADR 0033).
	Address string `json:"address,omitempty"`
	// external nodes: name + user-picked label + optional URL.
	Label ExternalLabel `json:"label,omitempty"`
}

type Node struct {
	ID       string   `json:"id"`
	Type     NodeType `json:"type"`
	Position Position `json:"position"`
	Data     NodeData `json:"data"`
}

type EdgeData struct {
	Kind RelationKind `json:"kind"`
}

// Edge documents a descriptive relation between two nodes (ADR 0033); it has no behavioral effect on real wiring.
type Edge struct {
	ID     string   `json:"id"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Type   string   `json:"type"`
	Data   EdgeData `json:"data"`
}

// Viewport is where the owner last left the camera: pan offset and zoom, saved with the map so a reload
// opens the canvas where it was.
type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

// Canvas is the workspace infrastructure map (ADR 0033): a descriptive model, not a controller.
type Canvas struct {
	SchemaVersion int       `json:"schema_version"`
	Nodes         []Node    `json:"nodes"`
	Edges         []Edge    `json:"edges"`
	Viewport      *Viewport `json:"viewport,omitempty"`
}
