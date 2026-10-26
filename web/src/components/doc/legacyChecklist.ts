import type { JSONContent } from "@tiptap/core";

// An item the editor saved before it had task items: a bullet whose text opens with "[ ] " or "[x] ".
const marker = /^\[([ xX])\](?: |$)/;

const markerOf = (item: JSONContent) => {
  const first = item.content?.[0]?.content?.[0];
  return first?.type === "text" ? marker.exec(first.text ?? "") : null;
};

const toTaskItem = (item: JSONContent, found: RegExpExecArray): JSONContent => {
  const [paragraph, ...rest] = item.content ?? [];
  const [text, ...inline] = paragraph?.content ?? [];
  const remaining = (text?.text ?? "").slice(found[0].length);
  const content = [...(remaining ? [{ ...text, text: remaining }] : []), ...inline];
  const opening = { ...paragraph, type: "paragraph", ...(content.length > 0 && { content }) };
  return { type: "taskItem", attrs: { checked: found[1] !== " " }, content: [opening, ...rest] };
};

// A bullet list becomes one list per run of task and plain items, the way the editor's markdown parser reads it.
const upgradeList = (list: JSONContent): JSONContent[] => {
  const items = upgradeBlocks(list.content ?? []);
  const runs: { task: boolean; items: JSONContent[] }[] = [];
  for (const item of items) {
    const found = markerOf(item);
    const last = runs.at(-1);
    const entry = found ? toTaskItem(item, found) : item;
    if (last && last.task === (found !== null)) last.items.push(entry);
    else runs.push({ task: found !== null, items: [entry] });
  }
  return runs.map((run) => (run.task ? { type: "taskList", content: run.items } : { ...list, content: run.items }));
};

const upgradeBlocks = (blocks: JSONContent[]): JSONContent[] =>
  blocks.flatMap((block) => {
    if (block.type === "bulletList") return upgradeList(block);
    if (!block.content) return [block];
    return [{ ...block, content: upgradeBlocks(block.content) }];
  });

/** Rewrites bullets saved as "[ ] text" into task items, so a body saved before task lists existed shows its boxes. */
export const upgradeChecklists = <T extends JSONContent | JSONContent[]>(doc: T): T =>
  (Array.isArray(doc) ? upgradeBlocks(doc) : { ...doc, content: upgradeBlocks(doc.content ?? []) }) as T;

/** The replacement for a bullet list in a live document, or null when it holds no such items. */
export const upgradedList = (list: JSONContent): JSONContent[] | null => {
  const next = upgradeList(list);
  return next.length === 1 && next[0]?.type === "bulletList" && JSON.stringify(next[0]) === JSON.stringify(list) ? null : next;
};
