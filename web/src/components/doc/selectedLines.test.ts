import { Editor } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import { afterEach, describe, expect, it } from "vitest";

import { isolateSelectedLines } from "@/components/doc/selectedLines";

const text = (t: string) => ({ type: "text", text: t });
const br = { type: "hardBreak" };

let editor: Editor | null = null;

afterEach(() => editor?.destroy());

// One paragraph whose lines are joined by line breaks: "TEST TEXT", a blank line, "Status and outcome".
const lineBrokenParagraph = () =>
  new Editor({
    extensions: [StarterKit.configure({ trailingNode: false })],
    content: {
      type: "doc",
      content: [{ type: "paragraph", content: [text("TEST TEXT"), br, br, text("Status and outcome")] }],
    },
  });

describe("isolateSelectedLines", () => {
  it("quotes only the selected line of a line-broken paragraph", () => {
    editor = lineBrokenParagraph();
    editor.chain().setTextSelection({ from: 1, to: 10 }).command(isolateSelectedLines).toggleBlockquote().run();

    expect(editor.getJSON().content).toEqual([
      { type: "blockquote", content: [{ type: "paragraph", content: [text("TEST TEXT")] }] },
      { type: "paragraph", content: [br, text("Status and outcome")] },
    ]);
  });

  it("makes a heading of a middle line without touching the lines around it", () => {
    editor = lineBrokenParagraph();
    const status = 12;
    editor.chain().setTextSelection({ from: status, to: status + 6 }).command(isolateSelectedLines).setHeading({ level: 2 }).run();

    expect(editor.getJSON().content).toEqual([
      { type: "paragraph", content: [text("TEST TEXT"), br] },
      { type: "heading", attrs: { level: 2 }, content: [text("Status and outcome")] },
    ]);
  });

  it("leaves a paragraph without line breaks whole", () => {
    editor = new Editor({ extensions: [StarterKit.configure({ trailingNode: false })], content: "<p>one line</p>" });
    editor.chain().setTextSelection({ from: 1, to: 4 }).command(isolateSelectedLines).toggleBulletList().run();

    expect(editor.getJSON().content).toEqual([
      { type: "bulletList", content: [{ type: "listItem", content: [{ type: "paragraph", content: [text("one line")] }] }] },
    ]);
  });
});
