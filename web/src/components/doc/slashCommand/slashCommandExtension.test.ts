import { Editor } from "@tiptap/core";
import { Table, TableCell, TableHeader, TableRow } from "@tiptap/extension-table";
import StarterKit from "@tiptap/starter-kit";
import { afterEach, describe, expect, it } from "vitest";

import { slashItemsFor } from "@/components/doc/slashCommand/slashCommandExtension";

let editor: Editor | null = null;

afterEach(() => editor?.destroy());

const positionOf = (e: Editor, text: string): number => {
  let found = -1;
  e.state.doc.descendants((node, pos) => {
    if (found < 0 && node.isText && node.text === text) found = pos + 1;
  });
  return found;
};

describe("slashItemsFor", () => {
  it("offers Table in a paragraph but not inside a table cell, where a nested table would flatten on save", () => {
    editor = new Editor({
      extensions: [StarterKit, Table, TableRow, TableHeader, TableCell],
      content: "<p>intro</p><table><tr><th><p>head</p></th></tr><tr><td><p>cell</p></td></tr></table>",
    });

    editor.commands.setTextSelection(positionOf(editor, "intro"));
    expect(slashItemsFor("", editor).map((i) => i.id)).toContain("table");

    editor.commands.setTextSelection(positionOf(editor, "cell"));
    const inCell = slashItemsFor("", editor).map((i) => i.id);
    expect(inCell).not.toContain("table");
    expect(inCell).toContain("bulletList");
  });
});
