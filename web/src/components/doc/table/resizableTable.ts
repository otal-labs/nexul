import { renderTableToMarkdown, Table } from "@tiptap/extension-table";
import { Plugin } from "@tiptap/pm/state";
import { columnResizingPluginKey } from "@tiptap/pm/tables";

import { cellMinWidth, freezeColumns, keyboardSizing, tableGrips } from "@/components/doc/table/tableSizing";

// Runs ahead of the table's own column resizing, which then starts its drag from the frozen width.
const columnFreezing = () =>
  new Plugin({
    props: {
      handleDOMEvents: {
        mousedown: (view, event) => {
          const handle = columnResizingPluginKey.getState(view.state)?.activeHandle ?? -1;
          if (view.editable && event.button === 0 && handle >= 0) freezeColumns(view, handle);
          return false;
        },
      },
    },
  });

export const ResizableTable = Table.extend({
  // The stock serializer leaves a pipe in cell text bare, so the next parse splits that cell in two.
  renderMarkdown: (node, h) =>
    renderTableToMarkdown(node, { ...h, renderChildren: (nodes, sep) => h.renderChildren(nodes, sep).replace(/\|/g, "\\|") }),
  addKeyboardShortcuts() {
    return { ...this.parent?.(), ...keyboardSizing };
  },
  addProseMirrorPlugins() {
    return [...(this.editor.isEditable ? [columnFreezing(), tableGrips()] : []), ...(this.parent?.() ?? [])];
  },
}).configure({ resizable: true, cellMinWidth });
