package richtext

import "regexp"

// legacyTaskMarker opens a bullet the editor saved before it had task items: "[ ] " or "[x] " ahead of the text.
var legacyTaskMarker = regexp.MustCompile(`^\[([ xX])\](?: |$)`)

// upgradeChecklists rewrites such bullets into task items, so a body saved before task lists existed renders as one.
// It mirrors web/src/components/doc/legacyChecklist.ts and returns new nodes, leaving its input untouched.
func upgradeChecklists(nodes []Node) []Node {
	out := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Type == "bulletList" {
			out = append(out, upgradeList(n)...)
			continue
		}
		if len(n.Content) > 0 {
			n.Content = upgradeChecklists(n.Content)
		}
		out = append(out, n)
	}
	return out
}

// upgradeList splits a bullet list into one list per run of task and plain items.
func upgradeList(list Node) []Node {
	type run struct {
		task  bool
		items []Node
	}
	var runs []run
	for _, item := range upgradeChecklists(list.Content) {
		found := legacyTaskMarker.FindStringSubmatch(openingText(item))
		task := found != nil
		if task {
			item = toTaskItem(item, found)
		}
		if n := len(runs); n > 0 && runs[n-1].task == task {
			runs[n-1].items = append(runs[n-1].items, item)
			continue
		}
		runs = append(runs, run{task: task, items: []Node{item}})
	}
	out := make([]Node, 0, len(runs))
	for _, r := range runs {
		if r.task {
			out = append(out, Node{Type: "taskList", Content: r.items})
			continue
		}
		plain := list
		plain.Content = r.items
		out = append(out, plain)
	}
	return out
}

func openingText(item Node) string {
	if len(item.Content) == 0 || len(item.Content[0].Content) == 0 {
		return ""
	}
	if first := item.Content[0].Content[0]; first.Type == "text" {
		return first.Text
	}
	return ""
}

func toTaskItem(item Node, found []string) Node {
	paragraph := item.Content[0]
	inline := paragraph.Content
	var content []Node
	if rest := inline[0].Text[len(found[0]):]; rest != "" {
		text := inline[0]
		text.Text = rest
		content = append(content, text)
	}
	content = append(content, inline[1:]...)
	paragraph.Content = content
	return Node{
		Type:    "taskItem",
		Attrs:   map[string]any{"checked": found[1] != " "},
		Content: append([]Node{paragraph}, item.Content[1:]...),
	}
}
