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

const LEADING_CD = /^cd +("[^"]+"|'[^']+'|\S+) *(?:&&|;) *([\s\S]*)$/;
const ABSOLUTE_PATH = /^(?:\/|~|[A-Za-z]:[\\/])/;

const unquote = (path: string): string => path.replace(/^(["'])([\s\S]*)\1$/, "$2");

const leadingCd = (command: string): { dir: string; rest: string } | null => {
  const match = LEADING_CD.exec(command);
  if (!match?.[1] || !match[2]) return null;
  return { dir: unquote(match[1]), rest: match[2] };
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
const readTarget = (summary: string, keys = READ_TARGET_KEYS): string | null => {
  try {
    const args: unknown = JSON.parse(summary);
    if (typeof args !== "object" || args === null) return null;
    const values = args as Record<string, unknown>;
    const key = keys.find((k) => typeof values[k] === "string" && values[k] !== "");
    return key === undefined ? null : (values[key] as string);
  } catch {
    return null;
  }
};

export const isMcpTool = (tool: string | undefined): boolean => tool !== undefined && mcpParts(tool) !== null;

const withoutFailed = (summary: string): string => (summary.endsWith(FAILED_SUFFIX) ? summary.slice(0, -FAILED_SUFFIX.length) : summary);

const shellCommand = (step: AgentStep): string => unwrapShellCommand(withoutFailed(step.summary));

// The run's own checkout: the absolute directory that the most commands start by changing into or below; null when none does.
export const runDirectory = (steps: AgentStep[]): string | null => {
  const dirs = steps
    .filter((step) => isCommandTool(step.tool))
    .map((step) => leadingCd(shellCommand(step))?.dir)
    .filter((dir): dir is string => dir !== undefined && ABSOLUTE_PATH.test(dir));
  let best: string | null = null;
  let bestCount = 0;
  for (const dir of new Set(dirs)) {
    const count = dirs.filter((d) => d === dir || d.startsWith(`${dir}/`)).length;
    if (count <= bestCount) continue;
    best = dir;
    bestCount = count;
  }
  return best;
};

// A command out of its shell wrapper, a leading `cd` into the run's directory cut, and one below it made relative.
export const commandText = (step: AgentStep, root: string | null): string => {
  const command = shellCommand(step);
  const cd = leadingCd(command);
  if (root === null || cd === null) return command;
  if (cd.dir === root) return cd.rest;
  if (cd.dir.startsWith(`${root}/`)) return `cd ${cd.dir.slice(root.length + 1)} && ${cd.rest}`;
  return command;
};

// One line for a step: `Server · tool` for an MCP call, the command alone out of its shell wrapper and its own leading
// `cd`, `tool: args` for any other tool with a file read's path or pattern for its arguments, the tool alone when its
// summary only repeats it, the summary alone for text, notes, and legacy lines.
export const stepLabel = (step: AgentStep): string => {
  const summary = withoutFailed(step.summary);
  if (!step.tool) return summary;
  if (isCommandTool(step.tool)) return commandText(step, runDirectory([step]));
  const mcp = mcpParts(step.tool);
  if (mcp) return `${mcp.server.charAt(0).toUpperCase()}${mcp.server.slice(1)}${SERVER_SEPARATOR}${mcp.name}`;
  if (summary === "" || summary === step.tool) return step.tool;
  const target = isFileReadTool(step.tool) ? readTarget(summary) : null;
  return `${step.tool}: ${target ?? summary}`;
};

const SEARCH_TOOLS = new Set(["Grep", "Glob", "LS"]);
const PATH_KEYS = ["file_path", "notebook_path", "path"];
const SEARCH_KEYS = ["pattern", "path"];

const isSearch = (step: AgentStep): boolean => SEARCH_TOOLS.has(step.tool ?? "");

// What a step acts on: the command as commandText reads it, the path or pattern a file tool names, else the summary.
export const stepTarget = (step: AgentStep, root: string | null = null): string => {
  const summary = withoutFailed(step.summary);
  if (isCommandTool(step.tool)) return commandText(step, root);
  if (!isFileTool(step.tool)) return summary;
  return readTarget(summary, isSearch(step) ? SEARCH_KEYS : PATH_KEYS) ?? summary;
};

export const fileName = (path: string): string => path.split(/[/\\]/).filter(Boolean).pop() ?? path;

// How a step reads in a turn: exploring files, editing them, running a command, calling any other tool, or the Agent saying something.
export type StepAct = "explore" | "edit" | "command" | "tool" | "say";

export const stepAct = (step: AgentStep): StepAct => {
  if (!isToolStep(step)) return "say";
  if (isCommandTool(step.tool)) return "command";
  if (isFileReadTool(step.tool)) return "explore";
  if (isFileTool(step.tool)) return "edit";
  return "tool";
};

// A step as a verb and its object, in the present while it runs ("Running", "go test") and the past once done; root
// is the run's directory, cut from the front of a command.
export const stepPhrase = (step: AgentStep, live: boolean, root: string | null = null): { verb: string; object: string } => {
  const act = stepAct(step);
  const target = stepTarget(step, root);
  if (act === "command") return { verb: live ? "Running" : "Ran", object: target };
  if (act === "explore" && isSearch(step)) return { verb: live ? "Searching for" : "Searched for", object: target };
  if (act === "explore") return { verb: live ? "Reading" : "Read", object: fileName(target) };
  if (act === "edit") return { verb: live ? "Editing" : "Edited", object: fileName(target) };
  if (act === "tool" && isMcpTool(step.tool)) return { verb: live ? "Calling" : "Called", object: stepLabel(step) };
  if (act === "tool") return { verb: live ? "Using" : "Used", object: step.tool ?? target };
  return { verb: "", object: target };
};

// The newest step's call while it runs, else the Agent thinking between steps.
export const liveStepLabel = (entries: AgentStep[]): string => {
  const last = entries[entries.length - 1];
  if (last?.kind !== "tool_call") return "Thinking";
  const { verb, object } = stepPhrase(last, true, runDirectory(entries));
  return `${verb} ${object}`;
};

// Neighbouring file reads and searches fold into one explore group, neighbouring edits into one edit group and
// neighbouring commands into one command group; every other step stands alone.
export interface StepGroup<T extends AgentStep> {
  act: StepAct;
  entries: T[];
}

const GROUPED_ACTS = new Set<StepAct>(["explore", "edit", "command"]);

export const groupSteps = <T extends AgentStep>(entries: T[]): StepGroup<T>[] => {
  const out: StepGroup<T>[] = [];
  for (const entry of entries) {
    const act = stepAct(entry);
    const last = out[out.length - 1];
    if (last?.act === act && GROUPED_ACTS.has(act)) {
      last.entries.push(entry);
      continue;
    }
    out.push({ act, entries: [entry] });
  }
  return out;
};

const plural = (n: number, noun: string, nouns = `${noun}s`): string => `${n} ${n === 1 ? noun : nouns}`;
const distinctFiles = (entries: AgentStep[]): number => new Set(entries.map((e) => fileName(stepTarget(e)))).size;

// "2 files, 1 search" for the reads and searches of an explore group or a whole turn.
export const exploreCount = (entries: AgentStep[]): string => {
  const explores = entries.filter((e) => stepAct(e) === "explore");
  const searches = explores.filter(isSearch).length;
  const files = distinctFiles(explores.filter((e) => !isSearch(e)));
  const parts: string[] = [];
  if (files > 0) parts.push(plural(files, "file"));
  if (searches > 0) parts.push(plural(searches, "search", "searches"));
  return parts.join(", ");
};

export const editCount = (entries: AgentStep[]): string => plural(distinctFiles(entries.filter((e) => stepAct(e) === "edit")), "file");

// The line over a group's names: "Explored 2 files, 1 search", "Edited 1 file", "Ran 3 commands"; present tense while one of them runs.
export const groupHeadline = (act: "explore" | "edit" | "command", entries: AgentStep[], running: boolean): string => {
  if (act === "explore") return `${running ? "Exploring" : "Explored"} ${exploreCount(entries)}`;
  if (act === "edit") return `${running ? "Editing" : "Edited"} ${editCount(entries)}`;
  return `${running ? "Running" : "Ran"} ${plural(entries.length, "command")}`;
};

// "explored 2 files, 1 search, edited 1 file, ran 2 commands"; the Agent's own sentences are not counted.
export const turnPhrase = (entries: AgentStep[]): string => {
  const parts: string[] = [];
  const explored = exploreCount(entries);
  const edits = entries.filter((e) => stepAct(e) === "edit");
  const commands = entries.filter((e) => stepAct(e) === "command").length;
  const tools = entries.filter((e) => stepAct(e) === "tool").length;
  if (explored !== "") parts.push(`explored ${explored}`);
  if (edits.length > 0) parts.push(`edited ${editCount(edits)}`);
  if (commands > 0) parts.push(`ran ${plural(commands, "command")}`);
  if (tools > 0) parts.push(`used ${plural(tools, "tool")}`);
  return parts.join(", ");
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

// How a turn stands, for the mark that leads its header: a result replaces its call (harness CallID), so a call left
// in a turn that is over never came back.
export type TurnOutcome = "running" | "failed" | "unfinished" | "done";

export const turnOutcome = (entries: AgentStep[], running: boolean): TurnOutcome => {
  if (running) return "running";
  if (entries.some(isFailedStep)) return "failed";
  if (entries.some((e) => e.kind === "tool_call")) return "unfinished";
  return "done";
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
