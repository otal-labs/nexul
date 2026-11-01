import type { JSONContent } from "@tiptap/core";

const isSpace = (node: JSONContent) => node.type === "hardBreak" || (node.type === "text" && !node.text?.trim());

const trimText = (node: JSONContent | undefined, trim: (text: string) => string): JSONContent[] => {
  if (!node) return [];
  if (node.type !== "text") return [node];
  const text = trim(node.text ?? "");
  return text ? [{ ...node, text }] : [];
};

// The line break that put an image on its own line goes with the image, so no paragraph keeps a dangling newline.
const trimRun = (run: JSONContent[]): JSONContent[] => {
  let start = 0;
  let end = run.length;
  while (start < end && isSpace(run[start]!)) start++;
  while (end > start && isSpace(run[end - 1]!)) end--;
  const kept = run.slice(start, end);
  if (kept.length === 1) return trimText(kept[0], (t) => t.trim());
  return [...trimText(kept[0], (t) => t.trimStart()), ...kept.slice(1, -1), ...trimText(kept.at(-1), (t) => t.trimEnd())];
};

const splitParagraph = (paragraph: JSONContent): JSONContent[] => {
  if (!paragraph.content?.some((node) => node.type === "image")) return [paragraph];
  const blocks: JSONContent[] = [];
  let run: JSONContent[] = [];
  const flush = () => {
    const content = trimRun(run);
    if (content.length > 0) blocks.push({ ...paragraph, content });
    run = [];
  };
  for (const node of paragraph.content) {
    if (node.type !== "image") {
      run.push(node);
      continue;
    }
    flush();
    blocks.push(node);
  }
  flush();
  return blocks;
};

const opensWithParagraph = new Set(["listItem", "taskItem"]);

const liftBlocks = (blocks: JSONContent[]): JSONContent[] =>
  blocks.flatMap((block) => {
    if (block.type === "paragraph") return splitParagraph(block);
    if (!block.content) return [block];
    const content = liftBlocks(block.content);
    const needsOpening = opensWithParagraph.has(block.type ?? "") && content[0]?.type !== "paragraph";
    return [{ ...block, content: needsOpening ? [{ type: "paragraph" }, ...content] : content }];
  });

// Markdown nests an image under a line of text inside that paragraph; the editor's image is a block, and a live room refuses an inline one.
export const liftImages = (doc: JSONContent): JSONContent => ({ ...doc, content: liftBlocks(doc.content ?? []) });
