// A deploy log line as both apps hold it: seq is the server's insert order, ts the runner's unix milliseconds.
interface LogLine {
  seq: number;
  ts: number;
}

// The newest line a cached log holds; the next batch is fetched as the lines after it.
export const lastLogSeq = (lines: readonly LogLine[]): number => lines.reduce((max, line) => Math.max(max, line.seq), 0);

// The cached log with a fetched batch merged in, in the server's order (time, then seq); a line fetched twice is kept once.
export const appendLogLines = <T extends LogLine>(lines: T[], added: readonly T[]): T[] => {
  const held = new Set(lines.map((line) => line.seq));
  const fresh = added.filter((line) => !held.has(line.seq));
  if (fresh.length === 0) return lines;
  return [...lines, ...fresh].sort((a, b) => a.ts - b.ts || a.seq - b.seq);
};
