package imports

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// notionBlock is one block of a page; Content is the payload under the key named by Type.
type notionBlock struct {
	ID          string
	Type        string
	HasChildren bool
	Parent      notionParent
	Content     notionContent
	// Children are read separately, 100 at a time.
	Children []notionBlock
}

// notionContent holds the fields of every block type this import reads; each type fills its own.
type notionContent struct {
	RichText        []notionRichText   `json:"rich_text"`
	Checked         bool               `json:"checked"`
	Language        string             `json:"language"`
	Caption         []notionRichText   `json:"caption"`
	Icon            *notionIcon        `json:"icon"`
	Title           string             `json:"title"`
	URL             string             `json:"url"`
	Expression      string             `json:"expression"`
	Type            string             `json:"type"`
	File            *notionFileURL     `json:"file"`
	External        *notionFileURL     `json:"external"`
	Name            string             `json:"name"`
	HasColumnHeader bool               `json:"has_column_header"`
	Cells           [][]notionRichText `json:"cells"`
	SyncedFrom      *struct {
		BlockID string `json:"block_id"`
	} `json:"synced_from"`
	PageID     string `json:"page_id"`
	DatabaseID string `json:"database_id"`
}

// UnmarshalJSON reads the common fields and the payload keyed by the block's type.
func (b *notionBlock) UnmarshalJSON(data []byte) error {
	var head struct {
		ID          string       `json:"id"`
		Type        string       `json:"type"`
		HasChildren bool         `json:"has_children"`
		Parent      notionParent `json:"parent"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	b.ID, b.Type, b.HasChildren, b.Parent = head.ID, head.Type, head.HasChildren, head.Parent
	raw, ok := all[head.Type]
	if !ok || string(raw) == "{}" {
		return nil
	}
	// A payload this import does not read, such as a column's, decodes to nothing rather than failing the page.
	_ = json.Unmarshal(raw, &b.Content)
	return nil
}

type notionIcon struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

type notionFileURL struct {
	URL string `json:"url"`
}

type notionRichText struct {
	Type        string `json:"type"`
	PlainText   string `json:"plain_text"`
	Href        string `json:"href"`
	Annotations struct {
		Bold          bool `json:"bold"`
		Italic        bool `json:"italic"`
		Strikethrough bool `json:"strikethrough"`
		Code          bool `json:"code"`
	} `json:"annotations"`
	Mention *struct {
		Type string `json:"type"`
		Page *struct {
			ID string `json:"id"`
		} `json:"page"`
	} `json:"mention"`
	Equation *struct {
		Expression string `json:"expression"`
	} `json:"equation"`
}

// notionFile is a file a block or property holds: hosted by Notion, which expires, or an external link.
type notionFile struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	File     *notionFileURL `json:"file"`
	External *notionFileURL `json:"external"`
}

type notionName struct {
	Name string `json:"name"`
}

// notionProperty is one property of a database row.
type notionProperty struct {
	Type        string           `json:"type"`
	Title       []notionRichText `json:"title"`
	RichText    []notionRichText `json:"rich_text"`
	Number      *float64         `json:"number"`
	Select      *notionName      `json:"select"`
	Status      *notionName      `json:"status"`
	MultiSelect []notionName     `json:"multi_select"`
	Date        *struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"date"`
	People      []notionName `json:"people"`
	Checkbox    bool         `json:"checkbox"`
	URL         *string      `json:"url"`
	Email       *string      `json:"email"`
	PhoneNumber *string      `json:"phone_number"`
	Files       []notionFile `json:"files"`
	Relation    []struct {
		ID string `json:"id"`
	} `json:"relation"`
	Formula *struct {
		Type    string   `json:"type"`
		String  *string  `json:"string"`
		Number  *float64 `json:"number"`
		Boolean *bool    `json:"boolean"`
	} `json:"formula"`
	CreatedTime    string      `json:"created_time"`
	LastEditedTime string      `json:"last_edited_time"`
	CreatedBy      *notionName `json:"created_by"`
	LastEditedBy   *notionName `json:"last_edited_by"`
	UniqueID       *struct {
		Prefix *string  `json:"prefix"`
		Number *float64 `json:"number"`
	} `json:"unique_id"`
	Rollup *struct {
		Type   string           `json:"type"`
		Number *float64         `json:"number"`
		Array  []notionProperty `json:"array"`
	} `json:"rollup"`
}

// notionTitle is a page's title, from its one title property.
func notionTitle(props map[string]notionProperty) string {
	for _, p := range props {
		if p.Type == "title" {
			return strings.TrimSpace(plainOf(p.Title))
		}
	}
	return ""
}

func plainOf(rts []notionRichText) string {
	var b strings.Builder
	for _, rt := range rts {
		b.WriteString(rt.PlainText)
	}
	return b.String()
}

// notionLink is the doc a Notion page becomes, for a link to it.
type notionLink struct {
	Href  string
	Title string
}

// notionWriter turns blocks into markdown: links to imported pages point at their docs, Notion-hosted files are
// collected to copy, and blocks with no markdown counterpart are counted.
type notionWriter struct {
	// link is the doc of an imported page or row, false for one outside the import.
	link func(id string) (notionLink, bool)
	// rows are the docs of a database's rows, empty for a view or a database outside the import.
	rows        func(databaseID string) []notionLink
	files       []string
	unsupported int
}

// listKind is the list a block continues, so consecutive items of one list sit on adjacent lines.
type listKind string

const (
	listNone   listKind = ""
	listBullet listKind = "bullet"
	listNumber listKind = "number"
	listTodo   listKind = "todo"
)

// blocks renders sibling blocks, blank-line separated except within one list.
func (w *notionWriter) blocks(bs []notionBlock) string {
	var b strings.Builder
	prev := listNone
	for _, blk := range bs {
		md, kind := w.block(blk)
		if md == "" {
			continue
		}
		if b.Len() > 0 {
			sep := "\n\n"
			if kind != listNone && kind == prev {
				sep = "\n"
			}
			b.WriteString(sep)
		}
		b.WriteString(md)
		prev = kind
	}
	return b.String()
}

func (w *notionWriter) block(blk notionBlock) (string, listKind) {
	for _, convert := range []func(notionBlock) (string, listKind, bool){w.textBlock, w.mediaBlock, w.containerBlock} {
		if md, kind, ok := convert(blk); ok {
			return md, kind
		}
	}
	w.unsupported++
	return "", listNone
}

// textBlock converts the blocks that are rich text: paragraphs, headings, list items, toggles, quotes and callouts.
func (w *notionWriter) textBlock(blk notionBlock) (string, listKind, bool) {
	c := blk.Content
	switch blk.Type {
	case "paragraph", "toggle":
		return joinBlocks(lineStart(w.text(c.RichText)), w.blocks(blk.Children)), listNone, true
	case "heading_1", "heading_2", "heading_3":
		level, _ := strconv.Atoi(strings.TrimPrefix(blk.Type, "heading_"))
		return joinBlocks(strings.Repeat("#", level)+" "+strings.ReplaceAll(w.text(c.RichText), "\\\n", " "), w.blocks(blk.Children)), listNone, true
	case "bulleted_list_item":
		return listItem("- ", w.text(c.RichText), w.blocks(blk.Children)), listBullet, true
	case "numbered_list_item":
		return listItem("1. ", w.text(c.RichText), w.blocks(blk.Children)), listNumber, true
	case "to_do":
		box := "- [ ] "
		if c.Checked {
			box = "- [x] "
		}
		return listItem(box, w.text(c.RichText), w.blocks(blk.Children)), listTodo, true
	case "quote":
		return quoted(joinBlocks(lineStart(w.text(c.RichText)), w.blocks(blk.Children))), listNone, true
	case "callout":
		text := w.text(c.RichText)
		if c.Icon != nil && c.Icon.Emoji != "" {
			text = c.Icon.Emoji + " " + text
		}
		return quoted(joinBlocks(text, w.blocks(blk.Children))), listNone, true
	}
	return "", listNone, false
}

// mediaBlock converts code, dividers, images, files and links; embeds and equations are counted as unsupported.
func (w *notionWriter) mediaBlock(blk notionBlock) (string, listKind, bool) {
	c := blk.Content
	switch blk.Type {
	case "code":
		return fenced(strings.ToLower(c.Language), plainOf(c.RichText)), listNone, true
	case "divider":
		return "---", listNone, true
	case "image":
		return "![" + escapeMarkdown(plainOf(c.Caption)) + "](" + w.fileURL(c) + ")", listNone, true
	case "file", "pdf", "video", "audio":
		u := w.fileURL(c)
		return "[" + escapeMarkdown(fileLabel(c, u)) + "](" + u + ")", listNone, true
	case "bookmark", "link_preview":
		return linkTo(firstNonEmpty(plainOf(c.Caption), c.URL), c.URL), listNone, true
	case "embed":
		w.unsupported++
		return linkTo(firstNonEmpty(plainOf(c.Caption), c.URL), c.URL), listNone, true
	case "equation":
		w.unsupported++
		return fenced("", c.Expression), listNone, true
	}
	return "", listNone, false
}

// containerBlock converts tables, child pages and databases, and flattens columns and synced blocks.
func (w *notionWriter) containerBlock(blk notionBlock) (string, listKind, bool) {
	c := blk.Content
	switch blk.Type {
	case "table":
		return w.table(blk), listNone, true
	case "child_page":
		return w.pageLink(blk.ID, c.Title), listNone, true
	case "link_to_page":
		return w.pageLink(firstNonEmpty(c.PageID, c.DatabaseID), ""), listNone, true
	case "child_database":
		return w.database(blk.ID, c.Title), listNone, true
	case "synced_block", "column_list":
		w.unsupported++
		return w.flat(blk.Children), listNone, true
	case "column":
		return w.flat(blk.Children), listNone, true
	}
	return "", listNone, false
}

// flat renders a container's blocks one after another, as a column's or a synced block's content.
func (w *notionWriter) flat(children []notionBlock) string {
	var parts []string
	for _, ch := range children {
		if ch.Type == "column" {
			parts = append(parts, w.blocks(ch.Children))
			continue
		}
		parts = append(parts, w.blocks([]notionBlock{ch}))
	}
	return joinBlocks(parts...)
}

// pageLink links a child page's doc; a page outside the import stays its title.
func (w *notionWriter) pageLink(id, title string) string {
	l, ok := w.link(id)
	if !ok {
		return lineStart(escapeMarkdown(title))
	}
	return "[" + escapeMarkdown(firstNonEmpty(l.Title, title, "Untitled")) + "](" + l.Href + ")"
}

// database lists its rows' docs under its title; a view of a database, or one outside the import, is skipped.
func (w *notionWriter) database(id, title string) string {
	rows := w.rows(id)
	if len(rows) == 0 {
		w.unsupported++
		return ""
	}
	items := make([]string, 0, len(rows))
	for _, r := range rows {
		items = append(items, "- ["+escapeMarkdown(firstNonEmpty(r.Title, "Untitled"))+"]("+r.Href+")")
	}
	return joinBlocks("**"+escapeMarkdown(firstNonEmpty(title, "Database"))+"**", strings.Join(items, "\n"))
}

func (w *notionWriter) table(blk notionBlock) string {
	var rows [][]string
	width := 0
	for _, r := range blk.Children {
		if r.Type != "table_row" {
			continue
		}
		cells := make([]string, 0, len(r.Content.Cells))
		for _, cell := range r.Content.Cells {
			cells = append(cells, cellText(w.text(cell)))
		}
		width = max(width, len(cells))
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return ""
	}
	return markdownTable(rows, width)
}

// markdownTable writes rows as a table whose first row is the header, every row padded to width.
func markdownTable(rows [][]string, width int) string {
	line := func(cells []string) string {
		for len(cells) < width {
			cells = append(cells, "")
		}
		return "| " + strings.Join(cells, " | ") + " |"
	}
	out := []string{line(rows[0]), line(slices.Repeat([]string{"---"}, width))}
	for _, r := range rows[1:] {
		out = append(out, line(r))
	}
	return strings.Join(out, "\n")
}

// fileURL is a file block's URL; a Notion-hosted one is collected to copy now, since it expires within the hour.
func (w *notionWriter) fileURL(c notionContent) string {
	if c.Type == "file" && c.File != nil {
		w.files = append(w.files, c.File.URL)
		return c.File.URL
	}
	if c.External != nil {
		return c.External.URL
	}
	return ""
}

func fileLabel(c notionContent, u string) string {
	if name := firstNonEmpty(strings.TrimSpace(plainOf(c.Caption)), strings.TrimSpace(c.Name)); name != "" {
		return name
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Path == "" {
		return "file"
	}
	return firstNonEmpty(strings.Trim(parsed.Path[strings.LastIndex(parsed.Path, "/")+1:], "/"), "file")
}

// text renders rich text with its marks; a link or mention of an imported page points at its doc.
func (w *notionWriter) text(rts []notionRichText) string {
	var b strings.Builder
	for _, rt := range rts {
		b.WriteString(w.span(rt))
	}
	return strings.ReplaceAll(b.String(), "\n", "\\\n")
}

func (w *notionWriter) span(rt notionRichText) string {
	if rt.Type == "equation" && rt.Equation != nil {
		w.unsupported++
		return "`" + strings.ReplaceAll(rt.Equation.Expression, "`", "'") + "`"
	}
	if rt.Type == "mention" && rt.Mention != nil && rt.Mention.Page != nil {
		return w.linked(escapeMarkdown(rt.PlainText), rt.Mention.Page.ID)
	}
	text := marked(rt)
	if rt.Href == "" {
		return text
	}
	if id, ok := notionPageID(rt.Href); ok {
		return w.linked(text, id)
	}
	return "[" + text + "](" + rt.Href + ")"
}

// linked links text to an imported page's doc, or else back to the page on Notion.
func (w *notionWriter) linked(text, id string) string {
	if l, ok := w.link(id); ok {
		return "[" + text + "](" + l.Href + ")"
	}
	return "[" + text + "](https://www.notion.so/" + strings.ReplaceAll(id, "-", "") + ")"
}

// marked wraps text in its marks, keeping its surrounding spaces outside them so the marks still parse.
func marked(rt notionRichText) string {
	raw := rt.PlainText
	core := strings.TrimSpace(raw)
	if core == "" {
		return raw
	}
	lead, trail := raw[:strings.Index(raw, core)], raw[strings.Index(raw, core)+len(core):]
	a := rt.Annotations
	if a.Code {
		return lead + "`" + strings.ReplaceAll(core, "`", "'") + "`" + trail
	}
	core = escapeMarkdown(core)
	if a.Bold {
		core = "**" + core + "**"
	}
	if a.Italic {
		core = "*" + core + "*"
	}
	if a.Strikethrough {
		core = "~~" + core + "~~"
	}
	return lead + core + trail
}

// notionIDPattern is a Notion id, with or without its dashes.
var notionIDPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}`)

// notionPageID is the page a Notion link points at: a relative /<id> or a notion.so or notion.site address.
func notionPageID(href string) (string, bool) {
	u, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	internal := host == "" && strings.HasPrefix(u.Path, "/")
	notion := host == "notion.so" || strings.HasSuffix(host, ".notion.so") || strings.HasSuffix(host, ".notion.site")
	if !internal && !notion {
		return "", false
	}
	ids := notionIDPattern.FindAllString(u.Path, -1)
	if len(ids) == 0 {
		return "", false
	}
	return canonicalNotionID(ids[len(ids)-1]), true
}

// canonicalNotionID is an id in Notion's dashed lowercase form, as its API returns ids.
func canonicalNotionID(id string) string {
	id = strings.ToLower(strings.ReplaceAll(id, "-", ""))
	if len(id) != 32 {
		return id
	}
	return id[:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:]
}

// rowTable is a row's properties as a two-column table, its title left out, in name order.
func (w *notionWriter) rowTable(props map[string]notionProperty) string {
	names := make([]string, 0, len(props))
	for name, p := range props {
		if p.Type != "title" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	slices.Sort(names)
	rows := [][]string{{"Property", "Value"}}
	for _, name := range names {
		rows = append(rows, []string{cellText(escapeMarkdown(name)), cellText(w.property(props[name]))})
	}
	return markdownTable(rows, 2)
}

func (w *notionWriter) property(p notionProperty) string {
	if f, ok := propertyText[p.Type]; ok {
		return f(p)
	}
	switch p.Type {
	case "rich_text":
		return w.text(p.RichText)
	case "files":
		return w.propertyFiles(p.Files)
	case "relation":
		return w.relations(p)
	case "rollup":
		return w.rollup(p)
	}
	return ""
}

// propertyText renders the property types that need nothing from the import.
var propertyText = map[string]func(notionProperty) string{
	"number":           func(p notionProperty) string { return number(p.Number) },
	"select":           func(p notionProperty) string { return nameOf(p.Select) },
	"status":           func(p notionProperty) string { return nameOf(p.Status) },
	"multi_select":     func(p notionProperty) string { return names(p.MultiSelect) },
	"people":           func(p notionProperty) string { return names(p.People) },
	"date":             dateText,
	"checkbox":         func(p notionProperty) string { return yesNo(p.Checkbox) },
	"url":              func(p notionProperty) string { return linkIfAny(p.URL) },
	"email":            func(p notionProperty) string { return escapeMarkdown(deref(p.Email)) },
	"phone_number":     func(p notionProperty) string { return escapeMarkdown(deref(p.PhoneNumber)) },
	"formula":          formula,
	"created_time":     func(p notionProperty) string { return p.CreatedTime },
	"last_edited_time": func(p notionProperty) string { return p.LastEditedTime },
	"created_by":       func(p notionProperty) string { return nameOf(p.CreatedBy) },
	"last_edited_by":   func(p notionProperty) string { return nameOf(p.LastEditedBy) },
	"unique_id":        uniqueID,
}

func dateText(p notionProperty) string {
	if p.Date == nil {
		return ""
	}
	return strings.TrimSuffix(p.Date.Start+" → "+p.Date.End, " → ")
}

func uniqueID(p notionProperty) string {
	if p.UniqueID == nil {
		return ""
	}
	return strings.TrimPrefix(deref(p.UniqueID.Prefix)+"-"+number(p.UniqueID.Number), "-")
}

func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func (w *notionWriter) propertyFiles(files []notionFile) string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		c := notionContent{Type: f.Type, File: f.File, External: f.External, Name: f.Name}
		u := w.fileURL(c)
		out = append(out, "["+escapeMarkdown(fileLabel(c, u))+"]("+u+")")
	}
	return strings.Join(out, ", ")
}

func (w *notionWriter) relations(p notionProperty) string {
	out := make([]string, 0, len(p.Relation))
	for _, r := range p.Relation {
		l, ok := w.link(r.ID)
		if !ok {
			continue
		}
		out = append(out, "["+escapeMarkdown(firstNonEmpty(l.Title, "Untitled"))+"]("+l.Href+")")
	}
	return strings.Join(out, ", ")
}

func (w *notionWriter) rollup(p notionProperty) string {
	if p.Rollup == nil {
		return ""
	}
	if p.Rollup.Type == "number" {
		return number(p.Rollup.Number)
	}
	if p.Rollup.Type != "array" {
		return ""
	}
	out := make([]string, 0, len(p.Rollup.Array))
	for _, item := range p.Rollup.Array {
		if v := w.property(item); v != "" {
			out = append(out, v)
		}
	}
	return strings.Join(out, ", ")
}

func formula(p notionProperty) string {
	f := p.Formula
	if f == nil {
		return ""
	}
	switch f.Type {
	case "string":
		return escapeMarkdown(deref(f.String))
	case "number":
		return number(f.Number)
	case "boolean":
		return yesNo(f.Boolean != nil && *f.Boolean)
	}
	return ""
}

func number(n *float64) string {
	if n == nil {
		return ""
	}
	return strconv.FormatFloat(*n, 'f', -1, 64)
}

func nameOf(n *notionName) string {
	if n == nil {
		return ""
	}
	return escapeMarkdown(n.Name)
}

func names(ns []notionName) string {
	out := make([]string, 0, len(ns))
	for _, n := range ns {
		out = append(out, escapeMarkdown(n.Name))
	}
	return strings.Join(out, ", ")
}

func linkIfAny(u *string) string {
	if u == nil || *u == "" {
		return ""
	}
	return linkTo(*u, *u)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func linkTo(label, u string) string {
	if u == "" {
		return escapeMarkdown(label)
	}
	return "[" + escapeMarkdown(label) + "](" + u + ")"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// markdownEscaper backslashes the characters that would turn plain text into marks or links.
var markdownEscaper = strings.NewReplacer(
	`\`, `\\`, "`", "\\`", `*`, `\*`, `_`, `\_`, `[`, `\[`, `]`, `\]`, `<`, `\<`, `>`, `\>`, `~`, `\~`, `|`, `\|`,
)

func escapeMarkdown(s string) string { return markdownEscaper.Replace(s) }

// orderedStart is text that would read as an ordered list item, such as "1. ".
var orderedStart = regexp.MustCompile(`^(\d+)([.)])(\s|$)`)

// lineStart escapes a leading character that would make plain text a heading or a list.
func lineStart(s string) string {
	if s != "" && strings.ContainsRune("#-+=", rune(s[0])) {
		return `\` + s
	}
	return orderedStart.ReplaceAllString(s, `$1\$2$3`)
}

// cellText keeps a table cell on one line.
func cellText(s string) string {
	return strings.TrimSpace(strings.NewReplacer("\\\n", " ", "\n", " ").Replace(s))
}

// listItem writes one item, its own later lines and its nested blocks indented under its marker.
func listItem(marker, text, children string) string {
	pad := strings.Repeat(" ", len(marker))
	first, rest, _ := strings.Cut(text, "\n")
	item := marker + first
	if rest != "" {
		item += "\n" + indent(rest, pad)
	}
	if children == "" {
		return item
	}
	return item + "\n" + indent(children, pad)
}

func indent(s, pad string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}

func quoted(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight("> "+l, " ")
	}
	return strings.Join(lines, "\n")
}

// fenced is a code block, its fence longer than any run of backticks inside it.
func fenced(lang, code string) string {
	fence := "```"
	for strings.Contains(code, fence) {
		fence += "`"
	}
	return fmt.Sprintf("%s%s\n%s\n%s", fence, strings.ReplaceAll(lang, "plain text", ""), code, fence)
}

func joinBlocks(parts ...string) string {
	kept := slices.DeleteFunc(slices.Clone(parts), func(p string) bool { return strings.TrimSpace(p) == "" })
	return strings.Join(kept, "\n\n")
}
