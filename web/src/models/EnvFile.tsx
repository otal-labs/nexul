export interface ParsedEnvFile {
  values: Record<string, string>;
  // 1-based numbers of the non-blank, non-comment lines that are not KEY=value.
  invalidLines: number[];
  duplicateKeys: string[];
}

const KEY_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;
const QUOTES = ["'", '"'];

const unquote = (value: string): string => {
  const first = value[0];
  if (value.length < 2 || first === undefined || !QUOTES.includes(first) || !value.endsWith(first)) return value;
  return value.slice(1, -1);
};

// Escapes stay literal on purpose: a value typed as \n is stored as backslash-n, never a newline.
export const parseEnvFile = (text: string): ParsedEnvFile => {
  const values: Record<string, string> = {};
  const invalidLines: number[] = [];
  const duplicateKeys: string[] = [];
  text.split(/\r?\n/).forEach((raw, index) => {
    const line = raw.trim();
    if (!line || line.startsWith("#")) return;
    const assignment = line.replace(/^export\s+/, "");
    const eq = assignment.indexOf("=");
    const key = assignment.slice(0, Math.max(eq, 0)).trim();
    if (eq <= 0 || !KEY_PATTERN.test(key)) return void invalidLines.push(index + 1);
    if (key in values && !duplicateKeys.includes(key)) duplicateKeys.push(key);
    values[key] = unquote(assignment.slice(eq + 1).trim());
  });
  return { values, invalidLines, duplicateKeys };
};

// ponytail: a value that both starts with a quote and contains both quote kinds is written bare and will not round-trip.
const quoteIfNeeded = (value: string): string => {
  if (!/^\s|\s$|^["']/.test(value)) return value;
  if (!value.includes('"')) return `"${value}"`;
  return value.includes("'") ? value : `'${value}'`;
};

export const formatEnvFile = (values: Record<string, string>, keys: string[]): string =>
  keys.map((key) => `${key}=${quoteIfNeeded(values[key] ?? "")}`).join("\n");
