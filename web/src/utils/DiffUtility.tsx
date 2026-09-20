export type DiffLineType = "context" | "add" | "remove";

export interface DiffLine {
  type: DiffLineType;
  text: string;
}

// ponytail: O(n*m) LCS table, fine for short handler files; swap in a diff dependency if inputs grow large.
export const diffLines = (before: string, after: string): DiffLine[] => {
  const a = before.split("\n");
  const b = after.split("\n");
  const n = a.length;
  const m = b.length;

  const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i]![j] = a[i] === b[j] ? lcs[i + 1]![j + 1]! + 1 : Math.max(lcs[i + 1]![j]!, lcs[i]![j + 1]!);
    }
  }

  const out: DiffLine[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ type: "context", text: a[i]! });
      i++;
      j++;
      continue;
    }
    if (lcs[i + 1]![j]! >= lcs[i]![j + 1]!) {
      out.push({ type: "remove", text: a[i]! });
      i++;
      continue;
    }
    out.push({ type: "add", text: b[j]! });
    j++;
  }
  while (i < n) {
    out.push({ type: "remove", text: a[i]! });
    i++;
  }
  while (j < m) {
    out.push({ type: "add", text: b[j]! });
    j++;
  }
  return out;
};
