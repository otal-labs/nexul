import { Editor } from "@tiptap/core";
import { afterEach, describe, expect, it } from "vitest";

import { buildEditorExtensions } from "@/components/doc/mention/editorExtensions";
import css from "@/index.css?raw";

// jsdom applies no stylesheet, so the guard runs index.css's own hide selector against the editor's real DOM.
const hideSelector = css.match(/([^{}]*ProseMirror-trailingBreak:only-child\))\s*\{\s*display: none;/)?.[1]?.trim() ?? "";

let editor: Editor | null = null;

afterEach(() => {
  editor?.destroy();
  document.body.innerHTML = "";
});

const trailingLineAfter = (html: string): HTMLElement => {
  const host = document.createElement("div");
  host.className = "prose-rich";
  document.body.append(host);
  editor = new Editor({ element: host, extensions: buildEditorExtensions() });
  editor.commands.setContent(html);
  const last = editor.view.dom.lastElementChild;
  if (!(last instanceof HTMLElement) || last.tagName !== "P") throw new Error("the editor kept no line after the last block");
  return last;
};

describe("the empty line after a body's last block", () => {
  it("stays after a closing table, the only line a click below it reaches to write or add a second table", () => {
    expect(hideSelector).not.toBe("");
    const line = trailingLineAfter("<p>intro</p><table><tr><th><p>A</p></th></tr><tr><td><p>1</p></td></tr></table>");

    expect(line.matches(hideSelector)).toBe(false);
  });

  it("hides after a closing code block while the editor is not focused", () => {
    const line = trailingLineAfter("<p>intro</p><pre><code>run</code></pre>");

    expect(line.matches(hideSelector)).toBe(true);
  });
});
