import type { KeyboardShortcutCommand } from "@tiptap/core";
import { Plugin } from "@tiptap/pm/state";
import { cellAround, columnResizingPluginKey, TableMap, updateColumnsOnResize } from "@tiptap/pm/tables";
import { Decoration, DecorationSet, type EditorView } from "@tiptap/pm/view";

import { minRowHeight, rowDrag } from "@/components/doc/table/resizableTableRow";

export const cellMinWidth = 48;
const keyStep = 16;

// Fixes every column at the width it shows, so the sized one can grow the table past the page instead of being squeezed.
export const freezeColumns = (view: EditorView, handle: number) => {
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

// The column a cell's right edge sizes: its last one, as the table's own drag does.
const columnAt = (view: EditorView, handle: number) => {
  const $cell = view.state.doc.resolve(handle);
  const table = $cell.node(-1);
  const map = TableMap.get(table);
  const col = map.colCount($cell.pos - $cell.start(-1)) + ($cell.nodeAfter?.attrs.colspan ?? 1) - 1;
  return { table, map, start: $cell.start(-1), col };
};

const columnWidth = (view: EditorView, handle: number) => {
  freezeColumns(view, handle);
  const widths = view.state.doc.nodeAt(handle)?.attrs.colwidth as number[] | null;
  return widths?.[widths.length - 1] ?? null;
};

const setColumnWidth = (view: EditorView, handle: number, width: number) => {
  const { table, map, start, col } = columnAt(view, handle);
  const tr = view.state.tr;
  new Set(map.map.filter((_, index) => index % map.width === col)).forEach((pos) => {
    const attrs = table.nodeAt(pos)!.attrs;
    const colwidth = [...(attrs.colwidth ?? Array<number>(attrs.colspan).fill(0))];
    colwidth[col - map.colCount(pos)] = width;
    tr.setNodeMarkup(start + pos, undefined, { ...attrs, colwidth });
  });
  view.dispatch(tr);
};

// Like the mouse drag: the column follows the finger and takes its width on release, with the ring line on its edge.
const columnDrag = (view: EditorView, handle: number, startX: number) => {
  const startWidth = columnWidth(view, handle);
  if (startWidth === null) return null;
  view.dispatch(view.state.tr.setMeta(columnResizingPluginKey, { setHandle: handle }));
  const width = (x: number) => Math.max(cellMinWidth, Math.round(startWidth + x - startX));
  const current = () => columnResizingPluginKey.getState(view.state)?.activeHandle ?? -1;
  return {
    place: (x: number) => {
      if (current() < 0) return;
      const { table, start, col } = columnAt(view, current());
      const dom = view.nodeDOM(start - 1);
      const tableDom = dom instanceof HTMLElement ? (dom.closest("table") ?? dom.querySelector("table")) : null;
      if (tableDom?.firstChild instanceof HTMLTableColElement) updateColumnsOnResize(table, tableDom.firstChild, tableDom, cellMinWidth, col, width(x));
    },
    finish: (x: number) => {
      const pos = current();
      view.dispatch(view.state.tr.setMeta(columnResizingPluginKey, { setHandle: -1 }));
      if (pos >= 0) setColumnWidth(view, pos, width(x));
    },
  };
};

const rowDragAt = (view: EditorView, cellPos: number) => {
  const row = view.nodeDOM(view.state.doc.resolve(cellPos).before());
  return row instanceof HTMLTableRowElement ? rowDrag(view, row) : null;
};

// Listens on the window: a size change rebuilds the cell and its grip mid-drag, and a finger stays captured by its first target.
const grip = (axis: "column" | "row") => (view: EditorView, getPos: () => number | undefined) => {
  const el = view.dom.ownerDocument.createElement("span");
  el.className = `table-grip table-grip-${axis} hit-area`;
  el.setAttribute("aria-hidden", "true");
  el.addEventListener("pointerdown", (event) => {
    const pos = getPos();
    if (event.button !== 0 || pos === undefined || !view.editable) return;
    const cellPos = pos - 1;
    const drag = axis === "column" ? columnDrag(view, cellPos, event.clientX) : rowDragAt(view, cellPos);
    if (!drag) return;
    event.preventDefault();
    const at = (e: PointerEvent) => (axis === "column" ? e.clientX : e.clientY);
    const win = view.dom.ownerDocument.defaultView ?? window;
    const place = (e: PointerEvent) => {
      if (e.pointerId === event.pointerId) drag.place(at(e));
    };
    const finish = (e: PointerEvent) => {
      if (e.pointerId !== event.pointerId) return;
      win.removeEventListener("pointermove", place);
      win.removeEventListener("pointerup", finish);
      win.removeEventListener("pointercancel", finish);
      drag.finish(at(e));
    };
    drag.place(at(event));
    win.addEventListener("pointermove", place);
    win.addEventListener("pointerup", finish);
    win.addEventListener("pointercancel", finish);
  });
  return el;
};

const columnGrip = grip("column");
const rowGrip = grip("row");

// The caret's cell carries a grip on its right and bottom edges; the stylesheet shows them only on a touch screen.
export const tableGrips = () =>
  new Plugin({
    props: {
      decorations: (state) => {
        const $cell = cellAround(state.selection.$from);
        if (!$cell) return null;
        const spec = { side: -1, ignoreSelection: true, stopEvent: () => true };
        return DecorationSet.create(state.doc, [
          Decoration.widget($cell.pos + 1, columnGrip, { ...spec, key: `column-grip-${$cell.pos}` }),
          Decoration.widget($cell.pos + 1, rowGrip, { ...spec, key: `row-grip-${$cell.pos}` }),
        ]);
      },
    },
  });

const resizeCaretColumn = (view: EditorView, direction: 1 | -1) => {
  const $cell = view.editable ? cellAround(view.state.selection.$from) : null;
  const width = $cell ? columnWidth(view, $cell.pos) : null;
  if (!$cell || width === null) return false;
  setColumnWidth(view, $cell.pos, Math.max(cellMinWidth, width + direction * keyStep));
  return true;
};

// From the height it shows, since a row's height is a minimum that taller text overrides.
const resizeCaretRow = (view: EditorView, direction: 1 | -1) => {
  const $cell = view.editable ? cellAround(view.state.selection.$from) : null;
  if (!$cell) return false;
  const pos = $cell.before();
  const row = view.nodeDOM(pos);
  if (!(row instanceof HTMLElement)) return false;
  const height = Math.round(row.getBoundingClientRect().height);
  view.dispatch(view.state.tr.setNodeAttribute(pos, "rowheight", Math.max(minRowHeight, height + direction * keyStep)));
  return true;
};

export const keyboardSizing: Record<string, KeyboardShortcutCommand> = {
  "Alt-Shift-ArrowRight": ({ editor }) => resizeCaretColumn(editor.view, 1),
  "Alt-Shift-ArrowLeft": ({ editor }) => resizeCaretColumn(editor.view, -1),
  "Alt-Shift-ArrowDown": ({ editor }) => resizeCaretRow(editor.view, 1),
  "Alt-Shift-ArrowUp": ({ editor }) => resizeCaretRow(editor.view, -1),
};
