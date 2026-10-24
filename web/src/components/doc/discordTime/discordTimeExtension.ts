import { Extension, type Editor } from "@tiptap/core";
import { Plugin, PluginKey, type EditorState } from "@tiptap/pm/state";
import { Decoration, DecorationSet } from "@tiptap/pm/view";

import { formatDiscordTime, fullDiscordTime, splitDiscordTimes, type DiscordTime } from "@nexul/client-core/discordTime";

interface FoundTime {
  from: number;
  to: number;
  time: DiscordTime;
}

const findTimes = (state: EditorState): FoundTime[] => {
  const found: FoundTime[] = [];
  state.doc.descendants((node, pos) => {
    if (node.type.spec.code) return false;
    if (!node.isText) return true;
    if (node.marks.some((mark) => mark.type.spec.code)) return false;
    let offset = 0;
    for (const part of splitDiscordTimes(node.text ?? "")) {
      const length = part.kind === "text" ? part.text.length : part.time.code.length;
      if (part.kind === "time") found.push({ from: pos + offset, to: pos + offset + length, time: part.time });
      offset += length;
    }
    return false;
  });
  return found;
};

const live = new Map<HTMLElement, DiscordTime>();
let ticker: ReturnType<typeof setInterval> | undefined;

// One clock for every relative chip on the page; a chip that left the document drops out on the next tick.
const tick = () => {
  const now = Date.now();
  for (const [label, time] of live) {
    if (!label.isConnected) {
      live.delete(label);
      continue;
    }
    label.textContent = formatDiscordTime(time, now);
  }
  if (live.size > 0) return;
  clearInterval(ticker);
  ticker = undefined;
};

const clockIcon = (): SVGSVGElement => {
  const ns = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(ns, "svg");
  const attrs = { viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round", "stroke-linejoin": "round", "aria-hidden": "true", class: "size-[1em] shrink-0" };
  for (const [key, value] of Object.entries(attrs)) svg.setAttribute(key, value);
  const circle = document.createElementNS(ns, "circle");
  for (const [key, value] of Object.entries({ cx: "12", cy: "12", r: "10" })) circle.setAttribute(key, value);
  const hands = document.createElementNS(ns, "path");
  hands.setAttribute("d", "M12 6v6l4 2");
  svg.append(circle, hands);
  return svg;
};

// The chat chip's markup, drawn as plain DOM inside the editor; a click on it opens the popover DiscordTimeLayer holds.
const chipElement = (time: DiscordTime): HTMLElement => {
  const chip = document.createElement("button");
  chip.type = "button";
  chip.className = "mention-chip";
  chip.contentEditable = "false";
  chip.title = fullDiscordTime(time.date);
  chip.dataset.discordTime = time.code;
  const label = document.createElement("time");
  label.className = "mention-chip__label";
  label.dateTime = time.date.toISOString();
  label.textContent = formatDiscordTime(time);
  chip.append(clockIcon(), label);
  if (time.style === "R") {
    live.set(label, time);
    ticker ??= setInterval(tick, 1_000);
  }
  return chip;
};

// The code under the caret stays raw so it can be edited; everywhere else the text is hidden behind its chip.
const decorate = (state: EditorState, editable: boolean): DecorationSet => {
  const { from: selFrom, to: selTo } = state.selection;
  const decorations = findTimes(state)
    .filter(({ from, to }) => !(editable && selFrom < to && selTo > from))
    .flatMap(({ from, to, time }) => [
      Decoration.widget(from, () => chipElement(time), { side: 1, key: `${from}:${time.code}`, ignoreSelection: true, stopEvent: () => true }),
      Decoration.inline(from, to, { class: "discord-time-raw" }),
    ]);
  return DecorationSet.create(state.doc, decorations);
};

const key = new PluginKey<DecorationSet>("discordTime");

// The body keeps the <t:…> code as plain text, so what is saved, synced and sent to agents is the code as written.
export const DiscordTimes = Extension.create({
  name: "discordTimes",
  addProseMirrorPlugins() {
    const editor: Editor = this.editor;
    return [
      new Plugin<DecorationSet>({
        key,
        state: {
          init: (_, state) => decorate(state, editor.isEditable),
          apply: (tr, old, _oldState, state) => (tr.docChanged || tr.selectionSet ? decorate(state, editor.isEditable) : old),
        },
        props: { decorations: (state) => key.getState(state) },
      }),
    ];
  },
});
