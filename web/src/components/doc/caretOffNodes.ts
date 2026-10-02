import { Extension } from "@tiptap/core";
import { NodeSelection, Plugin, Selection } from "@tiptap/pm/state";

// Loading a body maps the caret to its end, a node selection on a trailing image, so typing would replace the image.
export const CaretOffNodes = Extension.create({
  name: "caretOffNodes",
  addProseMirrorPlugins() {
    return [
      new Plugin({
        appendTransaction: (transactions, oldState, newState) => {
          const { selection } = newState;
          if (!(selection instanceof NodeSelection) || oldState.selection instanceof NodeSelection) return null;
          if (transactions.some((tr) => tr.selectionSet) || !transactions.some((tr) => tr.docChanged)) return null;
          const caret = Selection.findFrom(selection.$from, -1, true) ?? Selection.findFrom(selection.$to, 1, true);
          return caret && newState.tr.setSelection(caret);
        },
      }),
    ];
  },
});
