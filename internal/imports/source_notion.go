package imports

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// SourceNotion imports the pages shared with a Notion internal integration as docs, one folder per root page.
const SourceNotion = "notion"

// MaxNotionDepth is how many levels below its root page one import reads.
const MaxNotionDepth = 24

// notionCheckpointEvery is how many pages go by between saved cursors.
const notionCheckpointEvery = 25

// notionFileHosts are where Notion serves the files people upload, as signed links that expire within the hour.
var notionFileHosts = []string{"prod-files-secure.s3.us-west-2.amazonaws.com", "s3.us-west-2.amazonaws.com", "file.notion.so"}

// NotionSource is the Notion pages source.
type NotionSource struct {
	client *NotionClient
}

var _ Source = (*NotionSource)(nil)

// NewNotionSource wires the Notion source over client.
func NewNotionSource(client *NotionClient) *NotionSource { return &NotionSource{client: client} }

// Info implements Source.
func (n *NotionSource) Info() Info { return Info{Title: "Notion", FileHosts: notionFileHosts} }

// Scan implements Source: one search counts the pages under the roots, without reading any page's content.
func (n *NotionSource) Scan(ctx context.Context, job Job) (Scan, error) {
	plan, err := n.plan(ctx, job)
	if err != nil {
		return Scan{}, err
	}
	return Scan{From: plan.describeRoots(), Docs: len(plan.order)}, nil
}

// Import implements Source: each page's blocks are read, converted and handed over at once, so the signed file links
// they hold are downloaded long before they expire.
func (n *NotionSource) Import(ctx context.Context, job Job, sink Sink) error {
	plan, err := n.plan(ctx, job)
	if err != nil {
		return err
	}
	if len(plan.order) > MaxDocs {
		return fmt.Errorf("%w: the pages under the roots now number %d, more than the %d one import takes", apperrs.ErrInvalid, len(plan.order), MaxDocs)
	}
	start := 0
	if i := slices.Index(plan.order, job.Cursor); i >= 0 {
		start = i + 1
	}
	for i := start; i < len(plan.order); i++ {
		if err := n.handOver(ctx, job.Credential, plan, plan.nodes[plan.order[i]], sink); err != nil {
			return err
		}
		if (i+1)%notionCheckpointEvery == 0 {
			if err := sink.Checkpoint(ctx, plan.order[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// handOver writes one page's doc, unless an import already brought it in, which costs no read of its blocks.
func (n *NotionSource) handOver(ctx context.Context, token string, plan *notionPlan, node *notionNode, sink Sink) error {
	skip, err := sink.Skip(ctx, KindDoc, node.id)
	if err != nil || skip {
		return err
	}
	d, err := n.doc(ctx, token, plan, node, sink)
	if errors.Is(err, apperrs.ErrNotFound) {
		return sink.Fail(ctx, KindDoc, node.id, node.title, fmt.Errorf("%w: the page is no longer shared with the integration", apperrs.ErrNotFound))
	}
	if err != nil {
		return err
	}
	return sink.Doc(ctx, d)
}

// doc reads one page's blocks and writes its doc: a row opens with its properties, and links to pages in the import
// point at their docs.
func (n *NotionSource) doc(ctx context.Context, token string, plan *notionPlan, node *notionNode, sink Sink) (Doc, error) {
	blocks, err := n.tree(ctx, token, node.id, 0)
	if err != nil {
		return Doc{}, err
	}
	w := &notionWriter{
		link: func(id string) (notionLink, bool) {
			target, ok := plan.nodes[canonicalNotionID(id)]
			if !ok || !target.imported {
				return notionLink{}, false
			}
			return notionLink{Href: sink.DocLink(target.id), Title: target.title}, true
		},
		rows: func(databaseID string) []notionLink {
			db, ok := plan.nodes[canonicalNotionID(databaseID)]
			if !ok || !db.database {
				return nil
			}
			var out []notionLink
			for _, id := range db.children {
				if row := plan.nodes[id]; row.imported {
					out = append(out, notionLink{Href: sink.DocLink(row.id), Title: row.title})
				}
			}
			return out
		},
	}
	var table string
	if node.row {
		table = w.rowTable(node.props)
	}
	body := joinBlocks(table, w.blocks(blocks))
	return Doc{
		SourceID: node.id, Folder: cmp.Or(plan.nodes[node.root].title, "Notion"), Title: node.title, Body: body,
		Files: w.files, Unsupported: w.unsupported,
	}, nil
}

// tree reads a block's children and theirs, down to MaxNotionDepth; a child page or database is its own doc, and a
// synced block's content is read from its original.
func (n *NotionSource) tree(ctx context.Context, token, id string, depth int) ([]notionBlock, error) {
	blocks, err := n.client.children(ctx, token, id)
	if err != nil {
		return nil, err
	}
	for i := range blocks {
		b := &blocks[i]
		if !b.HasChildren || b.Type == "child_page" || b.Type == "child_database" || depth >= MaxNotionDepth {
			continue
		}
		from := b.ID
		if b.Type == "synced_block" && b.Content.SyncedFrom != nil {
			from = b.Content.SyncedFrom.BlockID
		}
		b.Children, err = n.tree(ctx, token, from, depth+1)
		if errors.Is(err, apperrs.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
	}
	return blocks, nil
}

// notionNode is a page or database shared with the integration, placed in the tree.
type notionNode struct {
	id, title string
	database  bool
	// row is a page whose parent is a database.
	row      bool
	props    map[string]notionProperty
	created  time.Time
	parent   string
	children []string
	// root is the root this node is imported under, and imported is whether it is a page the import writes.
	root     string
	imported bool
}

// notionPlan is every shared page placed in its tree, and the pages under the roots in the order they import.
type notionPlan struct {
	nodes map[string]*notionNode
	roots []string
	order []string
}

func (p *notionPlan) describeRoots() string {
	out := make([]string, 0, len(p.roots))
	for _, id := range p.roots {
		out = append(out, cmp.Or(p.nodes[id].title, "Untitled")+" ("+id+")")
	}
	return strings.Join(out, ", ")
}

// plan finds every page shared with the integration, places each under its parent, and picks the roots: the pages
// job.From names, or else every shared page whose parent is not shared.
func (n *NotionSource) plan(ctx context.Context, job Job) (*notionPlan, error) {
	token := strings.TrimSpace(job.Credential)
	if token == "" {
		return nil, fmt.Errorf("%w: a Notion import needs the internal integration's secret as its credential", apperrs.ErrInvalid)
	}
	pages, err := n.client.searchPages(ctx, token)
	if err != nil {
		return nil, err
	}
	plan := &notionPlan{nodes: map[string]*notionNode{}}
	parents := map[string]notionParent{}
	for _, pg := range pages {
		if pg.InTrash {
			continue
		}
		id := canonicalNotionID(pg.ID)
		row := pg.Parent.Type == "database_id" || pg.Parent.Type == "data_source_id"
		plan.nodes[id] = &notionNode{id: id, title: notionTitle(pg.Properties), row: row, props: pg.Properties, created: pg.CreatedTime}
		parents[id] = pg.Parent
	}
	if err := n.addDatabases(ctx, token, plan, parents); err != nil {
		return nil, err
	}
	blocks := map[string]string{}
	for id, parent := range parents {
		pid, err := n.parentNode(ctx, token, parent, blocks)
		if err != nil {
			return nil, err
		}
		if _, ok := plan.nodes[pid]; ok && pid != id {
			plan.nodes[id].parent = pid
			plan.nodes[pid].children = append(plan.nodes[pid].children, id)
		}
	}
	for _, node := range plan.nodes {
		slices.SortFunc(node.children, func(a, b string) int {
			return cmp.Or(plan.nodes[a].created.Compare(plan.nodes[b].created), strings.Compare(a, b))
		})
	}
	if err := plan.pickRoots(job.From); err != nil {
		return nil, err
	}
	return plan, plan.walk()
}

// addDatabases reads the database each row sits in, so a database becomes a node between its page and its rows.
func (n *NotionSource) addDatabases(ctx context.Context, token string, plan *notionPlan, parents map[string]notionParent) error {
	for _, parent := range slices.Collect(maps.Values(parents)) {
		if parent.DatabaseID == "" {
			continue
		}
		id := canonicalNotionID(parent.DatabaseID)
		if _, ok := plan.nodes[id]; ok {
			continue
		}
		db, err := n.client.database(ctx, token, id)
		if errors.Is(err, apperrs.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		plan.nodes[id] = &notionNode{id: id, title: strings.TrimSpace(plainOf(db.Title)), database: true}
		parents[id] = db.Parent
	}
	return nil
}

// parentNode is the page or database a parent names; a block parent is followed up to the page it sits in, and a
// parent the integration cannot see is "".
func (n *NotionSource) parentNode(ctx context.Context, token string, parent notionParent, blocks map[string]string) (string, error) {
	for range MaxNotionDepth * 4 {
		switch {
		case parent.Type == "page_id":
			return canonicalNotionID(parent.PageID), nil
		case parent.DatabaseID != "":
			return canonicalNotionID(parent.DatabaseID), nil
		case parent.Type != "block_id":
			return "", nil
		}
		id := canonicalNotionID(parent.BlockID)
		if pid, ok := blocks[id]; ok {
			return pid, nil
		}
		b, err := n.client.block(ctx, token, id)
		if errors.Is(err, apperrs.ErrNotFound) {
			blocks[id] = ""
			return "", nil
		}
		if err != nil {
			return "", err
		}
		next := b.Parent
		if next.Type != "block_id" {
			pid, err := n.parentNode(ctx, token, next, blocks)
			blocks[id] = pid
			return pid, err
		}
		parent = next
	}
	return "", nil
}

// pickRoots takes the pages from names, by id or link, else every node whose parent is not shared.
func (p *notionPlan) pickRoots(from string) error {
	for _, token := range strings.FieldsFunc(from, func(r rune) bool { return r == ',' || r == '\n' || r == ' ' }) {
		path, _, _ := strings.Cut(token, "?")
		ids := notionIDPattern.FindAllString(path, -1)
		if len(ids) == 0 {
			continue
		}
		id := canonicalNotionID(ids[len(ids)-1])
		if _, ok := p.nodes[id]; !ok {
			return fmt.Errorf("%w: page %s is not shared with the integration; in Notion, add the integration to it under Connections", apperrs.ErrInvalid, id)
		}
		if !slices.Contains(p.roots, id) {
			p.roots = append(p.roots, id)
		}
	}
	if len(p.roots) > 0 {
		p.roots = slices.DeleteFunc(p.roots, p.underAnotherRoot)
		return nil
	}
	for id, node := range p.nodes {
		if node.parent == "" {
			p.roots = append(p.roots, id)
		}
	}
	slices.SortFunc(p.roots, func(a, b string) int {
		return cmp.Or(strings.Compare(p.nodes[a].title, p.nodes[b].title), strings.Compare(a, b))
	})
	if len(p.roots) == 0 {
		return fmt.Errorf("%w: nothing is shared with the integration yet; in Notion, add it to the pages to import under Connections", apperrs.ErrInvalid)
	}
	return nil
}

// underAnotherRoot reports whether a picked root sits below another picked root, whose folder already takes it.
func (p *notionPlan) underAnotherRoot(id string) bool {
	for at, steps := p.nodes[id].parent, 0; at != "" && steps <= MaxNotionDepth*4; at, steps = p.nodes[at].parent, steps+1 {
		if slices.Contains(p.roots, at) {
			return true
		}
	}
	return false
}

// walk orders the pages under each root, parents first, refusing a tree deeper than MaxNotionDepth.
func (p *notionPlan) walk() error {
	var visit func(id, root string, depth int) error
	visit = func(id, root string, depth int) error {
		node := p.nodes[id]
		if node.root != "" {
			return nil
		}
		if depth > MaxNotionDepth {
			return fmt.Errorf("%w: %s sits %d levels below %s, deeper than the %d one import reads; pick a lower root page", apperrs.ErrInvalid, cmp.Or(node.title, id), depth, cmp.Or(p.nodes[root].title, root), MaxNotionDepth)
		}
		node.root = root
		if !node.database {
			node.imported = true
			p.order = append(p.order, id)
		}
		for _, child := range node.children {
			if err := visit(child, root, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range p.roots {
		if err := visit(root, root, 0); err != nil {
			return err
		}
	}
	return nil
}
