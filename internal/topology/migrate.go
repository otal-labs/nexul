package topology

// Migrate upgrades a stored canvas to the current schema, returning a copy and whether it changed.
func Migrate(c *Canvas) (*Canvas, bool) {
	if c == nil || c.SchemaVersion == CurrentSchemaVersion {
		return c, false
	}
	out := *c
	out.SchemaVersion = CurrentSchemaVersion
	out.Nodes = make([]Node, len(c.Nodes))
	copy(out.Nodes, c.Nodes)
	out.Edges = make([]Edge, len(c.Edges))
	copy(out.Edges, c.Edges)
	changed := false
	for i := range out.Nodes {
		n := &out.Nodes[i]
		if n.Type == NodeService && n.Data.ServiceID == "" {
			n.Data.ServiceID = n.ID
			changed = true
		}
	}
	if !changed && c.SchemaVersion == 1 {
		changed = true
	}
	return &out, changed
}
