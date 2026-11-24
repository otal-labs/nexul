import { Editor } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import { afterEach, describe, expect, it, vi } from "vitest";

import { EditorLoopGuard, loopBatches } from "@/components/doc/editorLoopGuard";
import { enqueue } from "@/lib/browserLogs";

vi.mock("@/lib/browserLogs", () => ({ enqueue: vi.fn() }));

const nextTask = () => new Promise((resolve) => setTimeout(resolve));

let editor: Editor | null = null;
let extension: MutationObserver | null = null;

afterEach(() => {
  extension?.disconnect();
  editor?.destroy();
  document.body.innerHTML = "";
  vi.mocked(enqueue).mockClear();
});

const mount = () => {
  const host = document.body.appendChild(document.createElement("div"));
  editor = new Editor({ element: host, extensions: [StarterKit, EditorLoopGuard], content: "<p>Capstone</p>" });
  return editor;
};

describe("EditorLoopGuard", () => {
  it("breaks the loop a page-rewriting extension starts, logs what kept changing, and gives the editor back", async () => {
    const ed = mount();
    // Restyles the paragraph each time the editor redraws it, as a dark-mode extension does; it gives up after 2000 rounds.
    let rounds = 0;
    extension = new MutationObserver(() => {
      if (++rounds > 2_000) return;
      ed.view.dom.querySelector("p")?.setAttribute("style", `color: rgb(${rounds % 255}, 0, 0)`);
    });
    extension.observe(ed.view.dom, { childList: true, subtree: true });

    ed.view.dom.querySelector("p")?.setAttribute("style", "color: red");
    await nextTask();

    expect(rounds).toBeLessThan(2_000);
    expect(enqueue).toHaveBeenCalledTimes(1);
    const [level, message, attrs] = vi.mocked(enqueue).mock.calls[0]!;
    expect([level, message]).toEqual(["error", "doc editor mutation loop"]);
    expect(attrs).toHaveProperty("html_attrs");
    expect(Object.values(attrs ?? {}).join("\n")).toMatch(/attr style on <P>/);
    await nextTask();
    expect(ed.isEditable).toBe(true);
  });

  it("logs nothing for a burst of ordinary edits", async () => {
    const ed = mount();
    for (let i = 0; i < loopBatches * 2; i++) ed.commands.insertContent("x");
    await nextTask();
    expect(enqueue).not.toHaveBeenCalled();
    expect(ed.isEditable).toBe(true);
  });
});
