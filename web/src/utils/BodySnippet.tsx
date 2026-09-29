interface RichNode {
  text?: string;
  content?: RichNode[];
}

const plainText = (node: RichNode): string => node.text ?? (node.content ?? []).map(plainText).join("");

const firstLine = (body: string): string =>
  (body.split("\n").find((line) => line.trim() !== "") ?? "").trim().replace(/^[#>*\- ]+/, "");

// The client's copy of the server's richtext.Snippet: a structured body's first block with text, else a legacy body's first line.
export const bodySnippet = (body: string): string => {
  let root: unknown;
  try {
    root = JSON.parse(body);
  } catch {
    return firstLine(body);
  }
  if (typeof root !== "object" || root === null) return firstLine(body);
  const blocks = Array.isArray(root) ? (root as RichNode[]) : ((root as RichNode).content ?? []);
  for (const block of blocks) {
    const text = plainText(block).trim();
    if (text !== "") return text;
  }
  return "";
};
