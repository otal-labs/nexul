import { Editor, type JSONContent } from "@tiptap/core";
import { fireEvent } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { buildEditorExtensions } from "@/components/doc/mention/editorExtensions";

const editors: Editor[] = [];

const mount = () => {
  const element = document.body.appendChild(document.createElement("div"));
  const editor = new Editor({ element, extensions: buildEditorExtensions(), content: "| Item | Qty |\n| --- | --- |\n| Bolts | 4 |", contentType: "markdown" });
  editors.push(editor);
  const row = editor.view.dom.querySelectorAll("tr")[1]!;
  row.getBoundingClientRect = () => ({ top: 100, bottom: 140, left: 0, right: 300, width: 300, height: 40, x: 0, y: 100, toJSON: () => ({}) });
  return { editor, cell: row.querySelector("td")! };
};

const rowHeights = (editor: Editor) => (editor.getJSON().content?.[0]?.content as JSONContent[]).map((row) => row.attrs?.rowheight);

afterEach(() => {
  for (const editor of editors.splice(0)) {
    editor.view.dom.parentElement?.remove();
    editor.destroy();
  }
});

describe("ResizableTableRow", () => {
  it("gives a row the height its bottom edge is dragged to, in one edit on release", () => {
    const { editor, cell } = mount();
    fireEvent.mouseDown(cell, { clientY: 138, button: 0 });
    fireEvent.mouseMove(window, { clientY: 170 });
    expect(document.querySelector(".row-resize-guide")).toHaveStyle({ top: "170px" });
    expect(rowHeights(editor)).toEqual([null, null]);

    fireEvent.mouseUp(window, { clientY: 190 });
    expect(rowHeights(editor)).toEqual([null, 90]);
    expect(editor.view.dom.querySelectorAll("tr")[1]).toHaveStyle({ height: "90px" });
    expect(document.querySelector(".row-resize-guide")).toBeNull();
  });

  it("leaves a press inside the row to the text, and never sizes a row below the floor", () => {
    const { editor, cell } = mount();
    fireEvent.mouseDown(cell, { clientY: 120, button: 0 });
    expect(document.querySelector(".row-resize-guide")).toBeNull();

    fireEvent.mouseDown(cell, { clientY: 139, button: 0 });
    fireEvent.mouseUp(window, { clientY: 101 });
    expect(rowHeights(editor)).toEqual([null, 24]);
  });
});
