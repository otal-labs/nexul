import type { Editor } from "@tiptap/core";
import { BubbleMenu } from "@tiptap/react/menus";
import { Grid2x2XIcon } from "lucide-react";

import { BubbleButton, BubbleDivider } from "@/components/doc/BubbleButton";

interface TableBubbleMenuProps {
  editor: Editor;
}

// The table around the caret, so the bar sits under the table instead of over the rows being edited.
const caretTable = (editor: Editor) => {
  const { $from } = editor.state.selection;
  for (let depth = $from.depth; depth > 0; depth--) {
    if ($from.node(depth).type.name !== "table") continue;
    const dom = editor.view.nodeDOM($from.before(depth));
    if (!(dom instanceof HTMLElement)) return null;
    return { getBoundingClientRect: () => dom.getBoundingClientRect(), getClientRects: () => dom.getClientRects() };
  }
  return null;
};

const textButton = "w-auto px-2 font-mono text-xs";

// Row and column edits while the caret is in a table; a text selection gets the format bubble instead.
export const TableBubbleMenu = ({ editor }: TableBubbleMenuProps) => (
  <BubbleMenu
    editor={editor}
    className="format-bubble"
    options={{ placement: "bottom-start", offset: 8 }}
    getReferencedVirtualElement={() => caretTable(editor)}
    shouldShow={({ editor: e, view }) => e.isEditable && view.hasFocus() && e.state.selection.empty && e.isActive("table")}
  >
    <BubbleButton label="Add row below" className={textButton} onClick={() => editor.chain().focus().addRowAfter().run()}>
      + Row
    </BubbleButton>
    <BubbleButton label="Add column right" className={textButton} onClick={() => editor.chain().focus().addColumnAfter().run()}>
      + Column
    </BubbleButton>
    <BubbleDivider />
    <BubbleButton label="Delete row" className={textButton} onClick={() => editor.chain().focus().deleteRow().run()}>
      − Row
    </BubbleButton>
    <BubbleButton label="Delete column" className={textButton} onClick={() => editor.chain().focus().deleteColumn().run()}>
      − Column
    </BubbleButton>
    <BubbleDivider />
    <BubbleButton label="Delete table" onClick={() => editor.chain().focus().deleteTable().run()}>
      <Grid2x2XIcon className="size-3.5" />
    </BubbleButton>
  </BubbleMenu>
);
