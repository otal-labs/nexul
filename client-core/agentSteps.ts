// Mirrors internal/harness.ActivityEntry's fields both apps read: one step of an agent's conversation (a trail, a setup run, a hand-off).
export interface AgentStep {
  kind: string;
  call_id?: string;
  tool?: string;
  summary: string;
  detail?: string;
  at: string;
}

// The harness's built-in tools whose row reads the command or the path alone, as the harness itself labels them.
const COMMAND_TOOLS = new Set(["Bash", "Shell"]);
const FILE_CHANGE_TOOLS = new Set(["Edit", "Write", "MultiEdit", "NotebookEdit"]);
const FILE_READ_TOOLS = new Set(["Read", "Glob", "Grep", "LS", "NotebookRead"]);

export const isCommandTool = (tool: string | undefined): boolean => tool !== undefined && COMMAND_TOOLS.has(tool);
export const isFileReadTool = (tool: string | undefined): boolean => tool !== undefined && FILE_READ_TOOLS.has(tool);
export const isFileTool = (tool: string | undefined): boolean => tool !== undefined && (FILE_CHANGE_TOOLS.has(tool) || FILE_READ_TOOLS.has(tool));
export const isToolStep = (step: AgentStep): boolean => step.kind === "tool_call" || step.kind === "tool_result";

// Mirrors the " · failed" the harness appends to a failed tool's summary; the row shows failure on its icon instead.
export const FAILED_SUFFIX = " · failed";

export const isFailedStep = (step: AgentStep): boolean => step.kind === "tool_result" && step.summary.endsWith(FAILED_SUFFIX);

const SHELL_WRAPPER = /^(?:\S*\/)?(?:bash|sh|zsh) +-l?c +/;

// The command a `/bin/bash -lc "…"` style wrapper runs, outer quotes off; a summary cut short keeps its open quote's text.
export const unwrapShellCommand = (command: string): string => {
  const match = SHELL_WRAPPER.exec(command);
  if (!match) return command;
  const inner = command.slice(match[0].length).trim();
  const quote = inner.charAt(0);
  if (quote !== '"' && quote !== "'") return inner;
  const closed = inner.length > 1 && inner.endsWith(quote);
  return (closed ? inner.slice(1, -1) : inner.slice(1)).trim();
};

const MCP_TOOL_NAME = /^mcp__(.+?)__(.+)$/i;
const SERVER_SEPARATOR = " · ";

// The server and tool of an MCP call: Claude names it mcp__<server>__<tool>, Codex arrives titled "<server> · <tool>".
const mcpParts = (tool: string): { server: string; name: string } | null => {
  const match = MCP_TOOL_NAME.exec(tool);
  if (match?.[1] && match[2]) return { server: match[1], name: match[2] };
  const at = tool.indexOf(SERVER_SEPARATOR);
  if (at <= 0) return null;
  return { server: tool.slice(0, at), name: tool.slice(at + SERVER_SEPARATOR.length) };
};

const READ_TARGET_KEYS = ["file_path", "notebook_path", "pattern", "path"];

// The path or pattern a read-type tool's JSON arguments name; null when the summary is not whole JSON (a cut-short preview).
const readTarget = (summary: string): string | null => {
  try {
    const args: unknown = JSON.parse(summary);
    if (typeof args !== "object" || args === null) return null;
    const values = args as Record<string, unknown>;
    const key = READ_TARGET_KEYS.find((k) => typeof values[k] === "string" && values[k] !== "");
    return key === undefined ? null : (values[key] as string);
  } catch {
    return null;
  }
};

export const isMcpTool = (tool: string | undefined): boolean => tool !== undefined && mcpParts(tool) !== null;

// One line for a step: `Server · tool` for an MCP call, the command alone out of its shell wrapper, `tool: args` for
// any other tool with a file read's path or pattern for its arguments, the tool alone when its summary only repeats it, the summary alone for text, notes, and legacy lines.
export const stepLabel = (step: AgentStep): string => {
  const summary = step.summary.endsWith(FAILED_SUFFIX) ? step.summary.slice(0, -FAILED_SUFFIX.length) : step.summary;
  if (!step.tool) return summary;
  if (isCommandTool(step.tool)) return unwrapShellCommand(summary);
  const mcp = mcpParts(step.tool);
  if (mcp) return `${mcp.server.charAt(0).toUpperCase()}${mcp.server.slice(1)}${SERVER_SEPARATOR}${mcp.name}`;
  if (summary === "" || summary === step.tool) return step.tool;
  const target = isFileReadTool(step.tool) ? readTarget(summary) : null;
  return `${step.tool}: ${target ?? summary}`;
};

// A run of steps between two of the agent's messages, folded into one group, or one message read as prose.
// from is the run's start for the first turn; until is when the turn ended when that is later than its last row.
export type StepSegment<T extends AgentStep> =
  | { kind: "turn"; entries: T[]; running: boolean; from: string | null; until: string | null }
  | { kind: "reply"; entry: T };

// Each message is prose and the tools between two messages fold into a group; empty until the first step lands.
export const segmentSteps = <T extends AgentStep>(steps: T[], running: boolean): StepSegment<T>[] => {
  const out: StepSegment<T>[] = [];
  let entries: T[] = [];
  let from: string | null = null;
  for (const step of steps) {
    if (step.kind !== "text") {
      entries.push(step);
      continue;
    }
    if (entries.length > 0) out.push({ kind: "turn", entries, running: false, from, until: step.at });
    out.push({ kind: "reply", entry: step });
    entries = [];
    from = step.at;
  }
  if (entries.length > 0 || (running && steps.length > 0)) out.push({ kind: "turn", entries, running, from, until: null });
  return out;
};

export interface TurnCounts {
  tools: number;
  commands: number;
  updates: number;
}

// Tool rows, command rows, and the Agent's sentences between them; a call and its result are one row.
export const turnCounts = (entries: AgentStep[]): TurnCounts => {
  const counts: TurnCounts = { tools: 0, commands: 0, updates: 0 };
  for (const entry of entries) {
    if (entry.kind === "text") {
      counts.updates += 1;
      continue;
    }
    if (isCommandTool(entry.tool)) {
      counts.commands += 1;
      continue;
    }
    counts.tools += 1;
  }
  return counts;
};

const plural = (n: number, noun: string): string => `${n} ${noun}${n === 1 ? "" : "s"}`;

// "Used 12 tools, ran 5 commands, and received 1 update"; only the non-zero parts, in that order.
export const turnSummary = ({ tools, commands, updates }: TurnCounts): string => {
  const parts: string[] = [];
  if (tools > 0) parts.push(`Used ${plural(tools, "tool")}`);
  if (commands > 0) parts.push(`${parts.length === 0 ? "Ran" : "ran"} ${plural(commands, "command")}`);
  if (updates > 0) parts.push(`${parts.length === 0 ? "Received" : "received"} ${plural(updates, "update")}`);
  if (parts.length === 0) return "No steps yet";
  if (parts.length === 1) return parts[0]!;
  return `${parts.slice(0, -1).join(", ")}${parts.length > 2 ? "," : ""} and ${parts[parts.length - 1]}`;
};

const timestampMs = (iso: string): number | null => {
  if (iso === "" || iso.startsWith("0001-")) return null;
  const ms = new Date(iso).getTime();
  return Number.isNaN(ms) ? null : ms;
};

// When the turn's first step happened, for the running clock; null when no step carries a time.
export const turnStartMs = (entries: AgentStep[], from: string | null = null): number | null => {
  const fromMs = from === null ? null : timestampMs(from);
  if (fromMs !== null) return fromMs;
  for (const entry of entries) {
    const ms = timestampMs(entry.at);
    if (ms !== null) return ms;
  }
  return null;
};

// The run's start (from, when given) or the first step, to the turn's end (until when given, else its last step), for
// a finished turn's "Worked for"; null when the times are missing.
export const turnSpanSeconds = (entries: AgentStep[], until: string | null = null, from: string | null = null): number | null => {
  const start = (from === null ? null : timestampMs(from)) ?? turnStartMs(entries);
  const end = (until === null ? null : timestampMs(until)) ?? turnStartMs([...entries].reverse());
  if (start === null || end === null) return null;
  return Math.max(0, Math.round((end - start) / 1_000));
};

export const formatElapsed = (seconds: number): string => {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
};
