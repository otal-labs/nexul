import { renderTableToMarkdown, Table } from "@tiptap/extension-table";
import { Plugin } from "@tiptap/pm/state";
import { columnResizingPluginKey, TableMap } from "@tiptap/pm/tables";
import type { EditorView } from "@tiptap/pm/view";

// Fixes every column at the width it shows, so the dragged one can grow the table past the page instead of being squeezed.
const freezeColumns = (view: EditorView) => {
  const handle = columnResizingPluginKey.getState(view.state)?.activeHandle ?? -1;
  if (handle < 0) return;
  const $cell = view.state.doc.resolve(handle);
  const table = $cell.node(-1);
  const start = $cell.start(-1);
  const map = TableMap.get(table);
  const firstRow = map.map.slice(0, map.width);
  const widths = firstRow.map((pos) => {
    const dom = view.nodeDOM(start + pos);
    return dom instanceof HTMLElement ? Math.ceil(dom.getBoundingClientRect().width) : 0;
  });
  if (widths.some((width) => width === 0)) return;
  const tr = view.state.tr;
  const seen = new Set<number>();
  map.map.forEach((pos, index) => {
    if (seen.has(pos)) return;
    seen.add(pos);
    const cell = table.nodeAt(pos);
    if (!cell || cell.attrs.colwidth) return;
    const col = index % map.width;
    tr.setNodeMarkup(start + pos, undefined, { ...cell.attrs, colwidth: widths.slice(col, col + cell.attrs.colspan) });
  });
  if (!tr.docChanged) return;
  view.dispatch(tr.setMeta("addToHistory", false));
};

// Runs ahead of the table's own column resizing, which then starts its drag from the frozen width.
const columnFreezing = () =>
  new Plugin({
    props: {
      handleDOMEvents: {
        mousedown: (view, event) => {
          if (view.editable && event.button === 0) freezeColumns(view);
          return false;
        },
      },
    },
  });

export const ResizableTable = Table.extend({
  // The stock serializer leaves a pipe in cell text bare, so the next parse splits that cell in two.
  renderMarkdown: (node, h) =>
    renderTableToMarkdown(node, { ...h, renderChildren: (nodes, sep) => h.renderChildren(nodes, sep).replace(/\|/g, "\\|") }),
  addProseMirrorPlugins() {
    return [...(this.editor.isEditable ? [columnFreezing()] : []), ...(this.parent?.() ?? [])];
  },
}).configure({ resizable: true, cellMinWidth: 48 });
