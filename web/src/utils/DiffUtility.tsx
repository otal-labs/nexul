export type DiffLineType = "context" | "add" | "remove";

export interface DiffLine {
  type: DiffLineType;
  text: string;
}

export type DiffSegment = DiffLine;

export interface DiffStats {
  cells: number;
}

// ponytail: the LCS table is O(n*m) over what is left after trimming, so two long texts that differ throughout stay slow; swap in a diff dependency if that input appears.
const diffTokens = (a: string[], b: string[], stats?: DiffStats): DiffLine[] => {
  let start = 0;
  while (start < a.length && start < b.length && a[start] === b[start]) start++;
  let endA = a.length;
  let endB = b.length;
  while (endA > start && endB > start && a[endA - 1] === b[endB - 1]) {
    endA--;
    endB--;
  }

  const n = endA - start;
  const m = endB - start;
  if (stats) stats.cells += (n + 1) * (m + 1);

  const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i]![j] = a[start + i] === b[start + j] ? lcs[i + 1]![j + 1]! + 1 : Math.max(lcs[i + 1]![j]!, lcs[i]![j + 1]!);
    }
  }

  const out: DiffLine[] = a.slice(0, start).map((text) => ({ type: "context", text }));
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[start + i] === b[start + j]) {
      out.push({ type: "context", text: a[start + i]! });
      i++;
      j++;
      continue;
    }
    if (lcs[i + 1]![j]! >= lcs[i]![j + 1]!) {
      out.push({ type: "remove", text: a[start + i]! });
      i++;
      continue;
    }
    out.push({ type: "add", text: b[start + j]! });
    j++;
  }
  while (i < n) {
    out.push({ type: "remove", text: a[start + i]! });
    i++;
  }
  while (j < m) {
    out.push({ type: "add", text: b[start + j]! });
    j++;
  }
  for (const text of a.slice(endA)) out.push({ type: "context", text });
  return out;
};

export const diffLines = (before: string, after: string, stats?: DiffStats): DiffLine[] =>
  diffTokens(before.split("\n"), after.split("\n"), stats);

const wordTokens = (text: string): string[] => text.split(/(\s+)/).filter(Boolean);

// Whitespace kept between two changed words belongs to both sides, so a changed phrase reads as one del and one ins.
export const diffWords = (before: string, after: string, stats?: DiffStats): DiffSegment[] => {
  const ops = diffTokens(wordTokens(before), wordTokens(after), stats);
  const out: DiffSegment[] = [];
  let removed = "";
  let added = "";

  const push = (type: DiffLineType, text: string) => {
    const last = out[out.length - 1];
    if (last?.type === type) {
      last.text += text;
      return;
    }
    out.push({ type, text });
  };
  const flush = () => {
    if (removed) push("remove", removed);
    if (added) push("add", added);
    removed = "";
    added = "";
  };

  ops.forEach((op, k) => {
    const prev = ops[k - 1];
    const next = ops[k + 1];
    const bridge = op.type === "context" && !op.text.trim() && prev && prev.type !== "context" && next && next.type !== "context";
    if (op.type === "context" && !bridge) {
      flush();
      push("context", op.text);
      return;
    }
    if (op.type !== "add") removed += op.text;
    if (op.type !== "remove") added += op.text;
  });
  flush();
  return out;
};
