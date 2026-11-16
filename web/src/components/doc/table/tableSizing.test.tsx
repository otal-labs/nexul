import { Editor, type JSONContent } from "@tiptap/core";
import { fireEvent } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { buildEditorExtensions } from "@/components/doc/mention/editorExtensions";

const editors: Editor[] = [];

const rect = (width: number, height: number) => ({ width, height, top: 0, bottom: height, left: 0, right: width, x: 0, y: 0, toJSON: () => ({}) });

const mount = (content = "| Item | Bin | Qty |\n| --- | --- | --- |\n| Bolts | A-12 | 4 |") => {
  const element = document.body.appendChild(document.createElement("div"));
  const editor = new Editor({ element, extensions: buildEditorExtensions(), content, contentType: "markdown" });
  editors.push(editor);
  for (const row of editor.view.dom.querySelectorAll("tr")) {
    row.getBoundingClientRect = () => rect(454, 40);
    [...row.children].forEach((cell, col) => {
      cell.getBoundingClientRect = () => rect([120, 96, 238][col]!, 40);
    });
  }
  return editor;
};

// The caret in the body row's first cell.
const caretInBodyCell = (editor: Editor) => {
  let pos = 0;
  editor.state.doc.descendants((node, at) => {
    if (node.isText && node.text === "Bolts") pos = at + 1;
  });
  editor.commands.setTextSelection(pos);
};

const table = (editor: Editor) => editor.getJSON().content?.find((node) => node.type === "table")?.content as JSONContent[];
const widths = (editor: Editor) => table(editor).map((row) => row.content?.map((cell) => cell.attrs?.colwidth));
const rowHeights = (editor: Editor) => table(editor).map((row) => row.attrs?.rowheight);
const press = (editor: Editor, key: string) => fireEvent.keyDown(editor.view.dom, { key, altKey: true, shiftKey: true });

afterEach(() => {
  for (const editor of editors.splice(0)) {
    editor.view.dom.parentElement?.remove();
    editor.destroy();
  }
});

describe("table sizing from the keyboard", () => {
  it("leaves the keys alone outside a table and in a body that can't be edited", () => {
    const editor = mount("Just text\n\n| Item | Bin | Qty |\n| --- | --- | --- |\n| Bolts | A-12 | 4 |");
    editor.commands.setTextSelection(2);
    press(editor, "ArrowRight");
    expect(widths(editor)?.[0]).toEqual([null, null, null]);

    const readOnly = mount();
    caretInBodyCell(readOnly);
    readOnly.setEditable(false);
    press(readOnly, "ArrowDown");
    expect(rowHeights(readOnly)).toEqual([null, null]);
  });

  it("widens and narrows the caret's column from the width it shows, never under the floor", () => {
    const editor = mount();
    caretInBodyCell(editor);
    press(editor, "ArrowRight");
    expect(widths(editor)).toEqual([
      [[136], [96], [238]],
      [[136], [96], [238]],
    ]);

    for (let i = 0; i < 10; i++) press(editor, "ArrowLeft");
    expect(widths(editor)?.[1]?.[0]).toEqual([48]);
  });

  it("makes the caret's row taller and shorter from the height it shows, never under the floor", () => {
    const editor = mount();
    caretInBodyCell(editor);
    press(editor, "ArrowDown");
    expect(rowHeights(editor)).toEqual([null, 56]);

    press(editor, "ArrowUp");
    press(editor, "ArrowUp");
    expect(rowHeights(editor)).toEqual([null, 24]);
  });
});

describe("table sizing by touch", () => {
  it("gives the caret cell's column the width its grip is dragged to, in one edit on release", () => {
    const editor = mount();
    caretInBodyCell(editor);
    const grip = editor.view.dom.querySelector(".table-grip-column")!;
    fireEvent.pointerDown(grip, { pointerId: 1, button: 0, clientX: 120 });
    fireEvent.pointerMove(window, { pointerId: 2, clientX: 400 });
    fireEvent.pointerMove(window, { pointerId: 1, clientX: 180 });
    expect(widths(editor)?.[1]?.[0]).toEqual([120]);

    fireEvent.pointerUp(window, { pointerId: 1, clientX: 200 });
    expect(widths(editor)).toEqual([
      [[200], [96], [238]],
      [[200], [96], [238]],
    ]);
  });

  it("gives the caret cell's row the height its grip is dragged to", () => {
    const editor = mount();
    caretInBodyCell(editor);
    const grip = editor.view.dom.querySelector(".table-grip-row")!;
    fireEvent.pointerDown(grip, { pointerId: 1, button: 0, clientY: 40 });
    expect(document.querySelector(".row-resize-guide")).not.toBeNull();

    fireEvent.pointerUp(window, { pointerId: 1, clientY: 90 });
    expect(rowHeights(editor)).toEqual([null, 90]);
    expect(document.querySelector(".row-resize-guide")).toBeNull();
  });
});
