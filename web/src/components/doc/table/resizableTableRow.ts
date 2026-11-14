import { TableRow } from "@tiptap/extension-table";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";

const edgeHeight = 5;
const minRowHeight = 24;
const rowEdgeKey = new PluginKey<boolean>("rowResize");

// The row under the pointer and its position in the doc, when the pointer sits on that row's bottom edge.
const rowEdgeAt = (view: EditorView, event: MouseEvent) => {
  if (!(event.target instanceof Element)) return null;
  const row = event.target.closest("tr");
  if (!row || !view.dom.contains(row)) return null;
  const gap = row.getBoundingClientRect().bottom - event.clientY;
  if (gap < 0 || gap > edgeHeight) return null;
  return rowPos(view, row) === null ? null : row;
};

const rowPos = (view: EditorView, row: HTMLTableRowElement) => {
  const pos = view.posAtDOM(row, 0) - 1;
  return view.state.doc.nodeAt(pos)?.type.name === "tableRow" ? pos : null;
};

const setHover = (view: EditorView, onEdge: boolean) => {
  if (rowEdgeKey.getState(view.state) === onEdge) return;
  view.dispatch(view.state.tr.setMeta(rowEdgeKey, onEdge));
};

// Like a spreadsheet, a guide line follows the pointer and the row takes its height on release, so a drag is one edit.
const startDrag = (view: EditorView, event: MouseEvent) => {
  const row = view.editable && event.button === 0 ? rowEdgeAt(view, event) : null;
  if (!row) return false;
  event.preventDefault();
  const win = view.dom.ownerDocument.defaultView ?? window;
  const frame = (row.closest(".tableWrapper") ?? row).getBoundingClientRect();
  const guide = view.dom.ownerDocument.body.appendChild(view.dom.ownerDocument.createElement("div"));
  guide.className = "row-resize-guide";
  guide.style.left = `${frame.left}px`;
  guide.style.width = `${frame.width}px`;
  const height = (y: number) => Math.max(minRowHeight, Math.round(y - row.getBoundingClientRect().top));
  const place = (e: MouseEvent) => {
    guide.style.top = `${row.getBoundingClientRect().top + height(e.clientY)}px`;
  };
  const finish = (e: MouseEvent) => {
    win.removeEventListener("mousemove", place);
    win.removeEventListener("mouseup", finish);
    guide.remove();
    setHover(view, false);
    // Found again from the row's DOM, since a collaborator's edit during the drag may have moved it.
    const pos = row.isConnected ? rowPos(view, row) : null;
    if (pos === null) return;
    view.dispatch(view.state.tr.setNodeAttribute(pos, "rowheight", height(e.clientY)));
  };
  place(event);
  win.addEventListener("mousemove", place);
  win.addEventListener("mouseup", finish);
  return true;
};

const rowResizing = () =>
  new Plugin<boolean>({
    key: rowEdgeKey,
    state: {
      init: () => false,
      apply: (tr, onEdge) => (tr.getMeta(rowEdgeKey) as boolean | undefined) ?? onEdge,
    },
    props: {
      attributes: (state): Record<string, string> => (rowEdgeKey.getState(state) ? { class: "row-resize-cursor" } : {}),
      handleDOMEvents: {
        mousemove: (view, event) => {
          setHover(view, view.editable && rowEdgeAt(view, event) !== null);
          return false;
        },
        mouseleave: (view) => {
          setHover(view, false);
          return false;
        },
        mousedown: startDrag,
      },
    },
  });

// A row's height is its minimum, as in a spreadsheet: content taller than it still grows the row.
export const ResizableTableRow = TableRow.extend({
  addAttributes() {
    return {
      rowheight: {
        default: null,
        parseHTML: (element) => Number.parseInt(element.style.height, 10) || null,
        renderHTML: (attributes) => (attributes.rowheight ? { style: `height: ${attributes.rowheight}px` } : {}),
      },
    };
  },
  addProseMirrorPlugins() {
    return this.editor.isEditable ? [rowResizing()] : [];
  },
});
