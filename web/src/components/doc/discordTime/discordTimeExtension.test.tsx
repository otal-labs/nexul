import { Editor } from "@tiptap/core";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";

import { DiscordTimeLayer } from "@/components/doc/discordTime/DiscordTimeLayer";
import { buildEditorExtensions } from "@/components/doc/mention/editorExtensions";

const UNIX = 1792867165;
const editors: Editor[] = [];
const elements: HTMLElement[] = [];

const mount = (markdown: string, editable = true) => {
  const element = document.createElement("div");
  document.body.append(element);
  elements.push(element);
  const editor = new Editor({ element, extensions: buildEditorExtensions(), editable, content: markdown, contentType: "markdown" });
  editors.push(editor);
  return editor;
};

const chips = (editor: Editor) => [...editor.view.dom.querySelectorAll<HTMLElement>("[data-discord-time]")];
const hidden = (editor: Editor) => [...editor.view.dom.querySelectorAll(".discord-time-raw")].map((el) => el.textContent);

afterEach(() => {
  for (const editor of editors.splice(0)) editor.destroy();
  for (const element of elements.splice(0)) element.remove();
});

describe("DiscordTimes", () => {
  it("draws a chip over each code in prose, hiding the code it stands for", () => {
    const editor = mount(`Standup <t:${UNIX}:F>, retro <t:${UNIX}:R>`, false);
    expect(chips(editor).map((chip) => chip.dataset.discordTime)).toEqual([`<t:${UNIX}:F>`, `<t:${UNIX}:R>`]);
    expect(chips(editor)[0]?.querySelector("time")).toHaveAttribute("dateTime", "2026-10-24T18:39:25.000Z");
    expect(hidden(editor)).toEqual([`<t:${UNIX}:F>`, `<t:${UNIX}:R>`]);
  });

  it("leaves codes in inline code, code blocks and unknown styles as typed", () => {
    const editor = mount(`\`<t:${UNIX}:F>\` and <t:${UNIX}:x>\n\n\`\`\`\n<t:${UNIX}:F>\n\`\`\``, false);
    expect(chips(editor)).toEqual([]);
  });

  it("shows the raw code while the caret is inside it, and the chip again once it leaves", () => {
    const editor = mount(`Standup <t:${UNIX}:F> today`);
    const start = editor.state.doc.textContent.indexOf("<t:") + 1;
    editor.commands.setTextSelection(start + 3);
    expect(chips(editor)).toEqual([]);
    editor.commands.setTextSelection(start + `<t:${UNIX}:F>`.length);
    expect(chips(editor)).toHaveLength(1);
  });

  it("chips a code as soon as it is pasted, and saves the body with the code as written", () => {
    const editor = mount("Standup ");
    editor.commands.insertContentAt(editor.state.doc.content.size - 1, `<t:${UNIX}:F>`);
    expect(chips(editor)).toHaveLength(1);
    expect(editor.getJSON().content?.[0]?.content).toEqual([{ type: "text", text: `Standup <t:${UNIX}:F>` }]);
  });
});

describe("DiscordTimeLayer", () => {
  it("opens the timestamp's popover on a click of an editor chip", async () => {
    const user = userEvent.setup();
    render(
      <DiscordTimeLayer>
        <button type="button" data-discord-time={`<t:${UNIX}:F>`}>
          chip
        </button>
      </DiscordTimeLayer>,
    );
    await user.click(screen.getByRole("button", { name: "chip" }));
    expect(await screen.findByRole("button", { name: "Copy code" })).toBeInTheDocument();
    expect(screen.getByText(`<t:${UNIX}:F>`)).toBeInTheDocument();
  });
});
