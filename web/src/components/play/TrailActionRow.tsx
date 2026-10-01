import { ChevronRight, CircleHelp, Dot, FileText, Terminal, Wrench } from "lucide-react";

import { ToolRow, type ToolRowStatus } from "@/components/play/ToolRowVisual";
import { TrailStepDetail } from "@/components/play/TrailStepDetail";
import { isCommandTool, isFailedStep, isFileReadTool, isFileTool, isMcpTool, stepLabel, type ActivityEntry } from "@/models/Trail";

interface TrailActionRowProps {
  entry: ActivityEntry;
  index?: number;
  // live marks a step from a run still going, so an unanswered call spins.
  live?: boolean;
  // entrance plays the mount animation; off when a finished turn is expanded, so the rows don't fade in again.
  entrance?: boolean;
}

const iconClass = "size-4";

const isTool = (entry: ActivityEntry): boolean => entry.kind === "tool_call" || entry.kind === "tool_result";

// A terminal for a command, a file for a file read or change, a wrench for an MCP call or any other tool; the Agent's own
// sentence has none and reads as prose.
const KindIcon = ({ entry }: { entry: ActivityEntry }) => {
  const tool = isTool(entry);
  return (
    <>
      {tool && isCommandTool(entry.tool) && <Terminal className={iconClass} role="img" aria-label="command" />}
      {tool && isFileTool(entry.tool) && <FileText className={iconClass} role="img" aria-label={isFileReadTool(entry.tool) ? "file" : "file change"} />}
      {tool && !isCommandTool(entry.tool) && !isFileTool(entry.tool) && (
        <Wrench className={iconClass} role="img" aria-label={entry.kind === "tool_call" ? "tool call" : "tool result"} />
      )}
      {entry.kind === "question" && <CircleHelp className={iconClass} role="img" aria-label="question" />}
      {(entry.kind === "other" || entry.kind === "note") && <Dot className={iconClass} role="img" aria-label="step" />}
    </>
  );
};

const statusOf = (entry: ActivityEntry, live: boolean): ToolRowStatus => {
  if (isFailedStep(entry)) return "failed";
  if (entry.kind === "tool_call" && live) return "running";
  return "idle";
};

// One action of the Agent's turn as a row; a step with detail expands to its arguments, result, and time, or the full text.
export const TrailActionRow = ({ entry, index = 0, live = false, entrance = true }: TrailActionRowProps) => {
  const row = (trailing?: React.ReactNode) => (
    <ToolRow
      icon={entry.kind !== "text" && <KindIcon entry={entry} />}
      label={stepLabel(entry)}
      mono={(isTool(entry) && !isMcpTool(entry.tool)) || entry.kind === "question"}
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
          <summary className="cursor-pointer list-none rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring [&::-webkit-details-marker]:hidden">
            {row(
              <ChevronRight
                className="size-3 text-muted-foreground/70 transition-transform duration-150 ease-standard group-open:rotate-90"
                aria-hidden
              />,
            )}
          </summary>
          <TrailStepDetail detail={entry.detail} at={entry.at} />
        </details>
      )}
    </li>
  );
};
