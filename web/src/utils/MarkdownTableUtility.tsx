const FENCE = /^ {0,3}(`{3,}|~{3,})/;
const DELIMITER_CELL = /^\s*:?-+:?\s*$/;

// Cells of a pipe row, outer pipes dropped; an escaped \| stays inside its cell.
const cells = (line: string): string[] =>
  line
    .trim()
    .replace(/^\|/, "")
    .replace(/(?<!\\)\|$/, "")
    .split(/(?<!\\)\|/);

const unquote = (line: string): string => line.replace(/^\s*(>\s?)*/, "");

const isTableStart = (header: string, delimiter: string): boolean => {
  if (!header.includes("|")) return false;
  const head = cells(header);
  const rule = cells(delimiter);
  return rule.length === head.length && rule.every((cell) => DELIMITER_CELL.test(cell));
};

// Whether markdown holds a GFM table outside fenced code: a piped header row directly over a delimiter row.
export const hasMarkdownTable = (markdown: string): boolean => {
  const lines = markdown.split(/\r?\n/).map(unquote);
  let fence: string | null = null;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i] ?? "";
    const marker = FENCE.exec(line)?.[1];
    if (fence !== null) {
      if (marker && marker[0] === fence[0] && marker.length >= fence.length) fence = null;
      continue;
    }
    if (marker) {
      fence = marker;
      continue;
    }
    const next = lines[i + 1];
    if (next !== undefined && !FENCE.test(next) && isTableStart(line, next)) return true;
  }
  return false;
};
