package richtext

// KeepTableSizes copies prev's table sizes onto next's tables in order, since a markdown edit would drop them (ADR 0026).
func KeepTableSizes(prev, next string) string {
	from, err := UnmarshalJSON(prev)
	if err != nil {
		return next
	}
	to, err := UnmarshalJSON(next)
	if err != nil {
		return next
	}
	fromTables, toTables := tables(from.Content), tables(to.Content)
	for i := range min(len(fromTables), len(toTables)) {
		keepSizes(fromTables[i], toTables[i])
	}
	out, err := MarshalJSON(to)
	if err != nil {
		return next
	}
	return out
}

func tables(nodes []Node) []*Node {
	var found []*Node
	for i := range nodes {
		if nodes[i].Type == "table" {
			found = append(found, &nodes[i])
			continue
		}
		found = append(found, tables(nodes[i].Content)...)
	}
	return found
}

func keepSizes(from, to *Node) {
	for r := range min(len(from.Content), len(to.Content)) {
		if h, ok := from.Content[r].Attrs["rowheight"]; ok && h != nil {
			setAttr(&to.Content[r], "rowheight", h)
		}
	}
	if len(from.Content) == 0 || len(to.Content) == 0 || len(from.Content[0].Content) != len(to.Content[0].Content) {
		return
	}
	widths := from.Content[0].Content
	for r := range to.Content {
		cells := to.Content[r].Content
		for c := range min(len(widths), len(cells)) {
			if w, ok := widths[c].Attrs["colwidth"]; ok && w != nil {
				setAttr(&cells[c], "colwidth", w)
			}
		}
	}
}

func setAttr(n *Node, key string, value any) {
	if n.Attrs == nil {
		n.Attrs = map[string]any{}
	}
	n.Attrs[key] = value
}
