import { Brain, ChevronRight, CircleHelp, Dot, FileText, SquareTerminal, Wrench } from "lucide-react";

import { ToolRow, type ToolRowStatus } from "@/components/play/ToolRowVisual";
import { TrailStepDetail } from "@/components/play/TrailStepDetail";
import { isCommandTool, isFileTool, stepLabel, type ActivityEntry } from "@/models/Trail";

interface TrailActionRowProps {
  entry: ActivityEntry;
  index?: number;
  // live marks a step from a run still going, so an unanswered call spins instead of sitting as a dot.
  live?: boolean;
  // entrance plays the mount animation; off when a finished turn is expanded, so the rows don't fade in again.
  entrance?: boolean;
}

const FAILED_SUFFIX = " · failed";

const iconClass = "size-3.5";

// A brain for the Agent's own sentence, a terminal for a command, a file for a file change, a wrench for any other
// tool; the label names what the icon depicts.
const KindIcon = ({ entry }: { entry: ActivityEntry }) => {
  const isTool = entry.kind === "tool_call" || entry.kind === "tool_result";
  return (
    <>
      {entry.kind === "text" && <Brain className={iconClass} role="img" aria-label="reasoning" />}
      {isTool && isCommandTool(entry.tool) && <SquareTerminal className={iconClass} role="img" aria-label="command" />}
      {isTool && isFileTool(entry.tool) && <FileText className={iconClass} role="img" aria-label="file change" />}
      {isTool && !isCommandTool(entry.tool) && !isFileTool(entry.tool) && (
        <Wrench className={iconClass} role="img" aria-label={entry.kind === "tool_call" ? "tool call" : "tool result"} />
      )}
      {entry.kind === "question" && <CircleHelp className={iconClass} role="img" aria-label="question" />}
      {(entry.kind === "other" || entry.kind === "note") && <Dot className={iconClass} role="img" aria-label="step" />}
    </>
  );
};

const statusOf = (entry: ActivityEntry, live: boolean): ToolRowStatus => {
  if (entry.kind === "tool_result") return entry.summary.endsWith(FAILED_SUFFIX) ? "failed" : "done";
  if (entry.kind === "tool_call") return live ? "running" : "pending";
  if (entry.kind === "question") return live ? "pending" : "idle";
  return "idle";
};

// The one-line result preview a finished tool call appends after its arguments.
const resultPreview = (entry: ActivityEntry): string | undefined => {
  if (entry.kind !== "tool_result" || !entry.detail) return undefined;
  try {
    const parsed: unknown = JSON.parse(entry.detail);
    if (typeof parsed !== "object" || parsed === null || !("result" in parsed)) return undefined;
    const result = parsed.result;
    const text =
      typeof result === "object" && result !== null && "content" in result && typeof result.content === "string"
        ? result.content
        : JSON.stringify(result);
    return text.replace(/\s+/g, " ").trim().slice(0, 80);
  } catch {
    return undefined;
  }
};

const clock = (iso: string): string | undefined => {
  if (iso === "" || iso.startsWith("0001-")) return undefined;
  return new Date(iso).toLocaleTimeString([], { hour12: false });
};

// One action of the Agent's turn as a row; a step with detail expands to its arguments and result, or the full text.
export const TrailActionRow = ({ entry, index = 0, live = false, entrance = true }: TrailActionRowProps) => {
  const row = (trailing?: React.ReactNode) => (
    <ToolRow
      icon={<KindIcon entry={entry} />}
      label={stepLabel(entry)}
      mono={entry.kind === "tool_call" || entry.kind === "tool_result" || entry.kind === "question"}
      result={resultPreview(entry)}
      meta={clock(entry.at)}
      status={statusOf(entry, live)}
      index={index}
      trailing={trailing}
      entrance={entrance}
    />
  );

  return (
    <li className="list-none">
      {!entry.detail && row()}
      {entry.detail && (
        <details className="group">
          <summary className="cursor-pointer list-none rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-ring [&::-webkit-details-marker]:hidden">
            {row(
              <ChevronRight
                className="size-3.5 text-muted-foreground transition-transform duration-150 ease-standard group-open:rotate-90"
                aria-hidden
              />,
            )}
          </summary>
          <TrailStepDetail detail={entry.detail} />
        </details>
      )}
    </li>
  );
};
