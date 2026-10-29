import type { JSONContent } from "@tiptap/core";

import { diffLines, diffWords, type DiffSegment } from "@/utils/DiffUtility";
import { parseBodyToJSON } from "@/utils/RichtextUtility";

export type HistoryBlockKind = "heading" | "paragraph" | "bullet" | "ordered" | "todo" | "done" | "quote" | "code" | "other";

export interface HistoryBlock {
  kind: HistoryBlockKind;
  text: string;
  /** A heading's level; 0 on every other block. */
  level: number;
  /** How deep in lists the block sits. */
  depth: number;
  /** An ordered item's number, shown but never compared, so inserting an item doesn't mark every later one edited. */
  ordinal: number;
}

export type RowMark = "same" | "edited" | "added" | "removed";

export interface HistoryRow {
  block: HistoryBlock;
  mark: RowMark;
  /** The block's text as context, add and remove runs; an unchanged or plain block is one context run. */
  parts: DiffSegment[];
}

const LIST_KINDS: Record<string, HistoryBlockKind> = { bulletList: "bullet", orderedList: "ordered", taskList: "todo" };

const make = (kind: HistoryBlockKind, text: string, depth: number, extra: Partial<HistoryBlock> = {}): HistoryBlock => ({
  kind,
  text,
  level: 0,
  depth,
  ordinal: 0,
  ...extra,
});

const inlineText = (node: JSONContent): string => {
  if (node.type === "text") return node.text ?? "";
  if (node.type === "hardBreak") return " ";
  if (node.type === "mention") return `@${node.attrs?.label ?? node.attrs?.id ?? ""}`;
  return (node.content ?? []).map(inlineText).join("");
};

// A node holding blocks (a table, its rows) reads with a space between them; a node holding only inline content reads straight through.
const plainText = (node: JSONContent): string =>
  node.content?.some((child) => child.content) ? node.content.map(plainText).join(" ") : inlineText(node);

const pushList = (list: JSONContent, depth: number, out: HistoryBlock[]) => {
  const kind = LIST_KINDS[list.type ?? ""] ?? "bullet";
  let ordinal = list.attrs?.start ?? 1;
  for (const item of list.content ?? []) {
    const [first, ...rest] = item.content ?? [];
    const leads = !!first && !((first.type ?? "") in LIST_KINDS);
    const itemKind = kind === "todo" && item.attrs?.checked ? "done" : kind;
    out.push(make(itemKind, leads ? inlineText(first) : "", depth, { ordinal }));
    ordinal++;
    pushBlocks(leads ? rest : (item.content ?? []), depth + 1, out);
  }
};

const pushBlocks = (nodes: JSONContent[], depth: number, out: HistoryBlock[]) => {
  for (const node of nodes) {
    const type = node.type ?? "";
    if (type === "heading") {
      out.push(make("heading", inlineText(node), depth, { level: node.attrs?.level ?? 2 }));
      continue;
    }
    if (type === "paragraph") {
      out.push(make("paragraph", inlineText(node), depth));
      continue;
    }
    if (type in LIST_KINDS) {
      pushList(node, depth, out);
      continue;
    }
    if (type === "codeBlock") {
      out.push(make("code", inlineText(node), depth));
      continue;
    }
    if (type === "blockquote") {
      for (const child of node.content ?? []) out.push(make("quote", plainText(child), depth));
      continue;
    }
    out.push(make("other", plainText(node) || `[${type}]`, depth));
  }
};

export const bodyToBlocks = (body: string): HistoryBlock[] => {
  const out: HistoryBlock[] = [];
  pushBlocks(parseBodyToJSON(body).content ?? [], 0, out);
  return out;
};

const keyOf = (b: HistoryBlock) => `${b.kind}\u0001${b.level}\u0001${b.depth}\u0001${b.text}`.replaceAll("\n", "↵");

const row = (b: HistoryBlock, mark: RowMark, parts: DiffSegment[]): HistoryRow => ({ block: b, mark, parts });
const addedRow = (b: HistoryBlock) => row(b, "added", [{ type: "add", text: b.text }]);
const removedRow = (b: HistoryBlock) => row(b, "removed", [{ type: "remove", text: b.text }]);
export const plainRow = (b: HistoryBlock) => row(b, "same", [{ type: "context", text: b.text }]);

// Two blocks are one edited block, not a removal beside an addition, when at least this share of the longer text survives.
const SAME_BLOCK_SHARE = 0.4;
// An added block this many places past a removed one is the furthest it is looked for.
const LOOKAHEAD = 3;

const keptShare = (before: string, after: string, parts: DiffSegment[]) => {
  const longest = Math.max(before.length, after.length);
  if (longest === 0) return 1;
  return parts.filter((p) => p.type === "context").reduce((sum, p) => sum + p.text.length, 0) / longest;
};

// A run of removed and added blocks between two unchanged ones: each removed block pairs with the next similar added one, in order.
const pairRun = (removed: HistoryBlock[], added: HistoryBlock[]): HistoryRow[] => {
  const rows: HistoryRow[] = [];
  let next = 0;
  for (const old of removed) {
    let found: { at: number; parts: DiffSegment[] } | undefined;
    for (let at = next; at < Math.min(added.length, next + LOOKAHEAD) && !found; at++) {
      const parts = diffWords(old.text, added[at]!.text);
      if (keptShare(old.text, added[at]!.text, parts) >= SAME_BLOCK_SHARE) found = { at, parts };
    }
    if (!found) {
      rows.push(removedRow(old));
      continue;
    }
    for (; next < found.at; next++) rows.push(addedRow(added[next]!));
    rows.push(row(added[found.at]!, "edited", found.parts));
    next = found.at + 1;
  }
  for (; next < added.length; next++) rows.push(addedRow(added[next]!));
  return rows;
};

// Every block of the newer body in reading order, with removed blocks of the older one in the places they left.
export const diffBlocks = (before: HistoryBlock[], after: HistoryBlock[]): HistoryRow[] => {
  if (before.length === 0) return after.map(addedRow);
  if (after.length === 0) return before.map(removedRow);

  const rows: HistoryRow[] = [];
  let removed: HistoryBlock[] = [];
  let added: HistoryBlock[] = [];
  let b = 0;
  let a = 0;
  const flush = () => {
    rows.push(...pairRun(removed, added));
    removed = [];
    added = [];
  };

  for (const segment of diffLines(before.map(keyOf).join("\n"), after.map(keyOf).join("\n"))) {
    if (segment.type === "remove") {
      removed.push(before[b++]!);
      continue;
    }
    if (segment.type === "add") {
      added.push(after[a++]!);
      continue;
    }
    flush();
    b++;
    rows.push(plainRow(after[a++]!));
  }
  flush();
  return rows;
};

export interface HistoryRun {
  mark: RowMark;
  rows: HistoryRow[];
  /** Its 1-based place among the changed runs, which the stepper counts; 0 on an unchanged run. */
  change: number;
}

// One change is one run of adjacent blocks with the same mark, so the stepper's count is the number of bars on screen; a changed title is change 1 and the runs count on from firstChange.
export const groupRuns = (rows: HistoryRow[], firstChange = 1): HistoryRun[] => {
  const runs: HistoryRun[] = [];
  let changes = firstChange - 1;
  for (const row of rows) {
    const last = runs[runs.length - 1];
    if (last?.mark === row.mark) {
      last.rows.push(row);
      continue;
    }
    if (row.mark !== "same") changes++;
    runs.push({ mark: row.mark, rows: [row], change: row.mark === "same" ? 0 : changes });
  }
  return runs;
};
