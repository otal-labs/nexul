import {
  MAX_LOG_LINES,
  type ContainerLogLine,
  type ContainerLogs,
  type ContainerLogWireLine,
  type LogFilter,
} from "@/models/ContainerLog";
import { formatLogTimestamp } from "@/utils/DeployLogUtility";

// Output has no levels, so this is a text match (a logfmt or JSON level carries the bare word); stderr is no signal, some programs write all to it.
const ERROR_WORDS = /\b(?:error|fatal|panic|critical)\b/i;
const STACK_TRACE_START = /^\s+at\s|goroutine \d+ \[|Traceback|Exception/;

export const looksLikeError = (text: string): boolean => ERROR_WORDS.test(text) || STACK_TRACE_START.test(text);

export const filterLogLines = (lines: ContainerLogLine[], filter: LogFilter): ContainerLogLine[] =>
  filter === "errors" ? lines.filter((line) => looksLikeError(line.text)) : lines;

// Numbers new lines after the newest kept one and drops the oldest past the cap; seq never repeats within a view.
export const appendLogLines = (kept: ContainerLogLine[], incoming: ContainerLogWireLine[]): ContainerLogLine[] => {
  if (incoming.length === 0) return kept;
  let seq = (kept.at(-1)?.seq ?? -1) + 1;
  const added = incoming.map((l): ContainerLogLine => ({ seq: seq++, ts: l.ts, stream: l.stream, text: l.line }));
  return [...kept, ...added].slice(-MAX_LOG_LINES);
};

// What the block says in place of lines: the connection's state while nothing has arrived, else why the filter shows none.
export const emptyLogMessage = ({ lines, status, reason }: ContainerLogs, filter: LogFilter): string => {
  if (status === "forbidden") return "You can't read logs for this stack.";
  if (lines.length > 0 && filter === "errors") return "No error output";
  if (status === "connecting") return "Connecting…";
  if (status === "offline") return reason ? `Runner offline: ${reason}` : "Runner offline";
  return "No output yet";
};

// The daemon's own messages (a container that does not exist) arrive without a timestamp.
export const formatContainerLogTime = (line: ContainerLogLine): string => {
  const ms = Date.parse(line.ts);
  return Number.isNaN(ms) ? "" : formatLogTimestamp(ms);
};

export const containerLogToText = (lines: ContainerLogLine[]): string =>
  lines.map((line) => `${formatContainerLogTime(line)}  ${line.text}`).join("\n");

export const saveTextFile = (filename: string, text: string) => {
  const url = URL.createObjectURL(new Blob([text], { type: "text/plain" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  // Revoked on the next tick: revoking inside the click handler can cancel the download in some browsers.
  setTimeout(() => URL.revokeObjectURL(url), 0);
};

const CHUNK_LINES = 50;

// Buckets by seq, not by position, so trimming the oldest line or filtering never reshuffles the chunks in between.
export const chunkLogLines = (lines: ContainerLogLine[]): [number, ContainerLogLine[]][] => {
  const chunks = new Map<number, ContainerLogLine[]>();
  for (const line of lines) {
    const bucket = Math.floor(line.seq / CHUNK_LINES);
    const chunk = chunks.get(bucket);
    if (chunk) chunk.push(line);
    if (!chunk) chunks.set(bucket, [line]);
  }
  return [...chunks];
};
