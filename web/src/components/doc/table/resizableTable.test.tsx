import { Editor, type JSONContent } from "@tiptap/core";
import { columnResizingPluginKey } from "@tiptap/pm/tables";
import { fireEvent } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { buildEditorExtensions } from "@/components/doc/mention/editorExtensions";

const editors: Editor[] = [];

const mount = () => {
  const element = document.body.appendChild(document.createElement("div"));
  const editor = new Editor({ element, extensions: buildEditorExtensions(), content: "| Item | Bin | Qty |\n| --- | --- | --- |\n| Bolts | A-12 | 4 |", contentType: "markdown" });
  editors.push(editor);
  for (const row of editor.view.dom.querySelectorAll("tr")) {
    row.getBoundingClientRect = () => ({ width: 454, height: 34, top: 0, bottom: 34, left: 0, right: 454, x: 0, y: 0, toJSON: () => ({}) });
    [...row.children].forEach((cell, col) => {
      const width = [120, 96, 238][col]!;
      cell.getBoundingClientRect = () => ({ width, height: 34, top: 0, bottom: 34, left: 0, right: width, x: 0, y: 0, toJSON: () => ({}) });
    });
  }
  return editor;
};

const widths = (editor: Editor) =>
  (editor.getJSON().content?.[0]?.content as JSONContent[]).map((row) => row.content?.map((cell) => cell.attrs?.colwidth));

afterEach(() => {
  for (const editor of editors.splice(0)) {
    editor.view.dom.parentElement?.remove();
    editor.destroy();
  }
});

describe("ResizableTable", () => {
  it("fixes every column at its shown width when one is dragged, so the dragged one keeps the width it gets", () => {
    const editor = mount();
    const firstHeader = editor.state.doc.resolve(3).before();
    editor.view.dispatch(editor.state.tr.setMeta(columnResizingPluginKey, { setHandle: firstHeader }));

    fireEvent.mouseDown(editor.view.dom.querySelector("th")!, { clientX: 120, clientY: 10, button: 0 });
    fireEvent.mouseUp(window, { clientX: 300 });

    expect(widths(editor)).toEqual([
      [[300], [96], [238]],
      [[300], [96], [238]],
    ]);
  });

  it("leaves an unsized table alone when no column edge is under the pointer", () => {
    const editor = mount();
    fireEvent.mouseDown(editor.view.dom.querySelector("td")!, { clientX: 40, clientY: 10, button: 0 });
    expect(widths(editor)).toEqual([
      [null, null, null],
      [null, null, null],
    ]);
  });
});
