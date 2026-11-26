import { Fragment, memo } from "react";

import { cn } from "@/lib/utils";
import type { ContainerLogLine } from "@/models/ContainerLog";
import { breakableParts, formatContainerLogTime, logLineParts, type LogLevelTone } from "@/utils/ContainerLogUtility";

interface LogLineRowProps {
  line: ContainerLogLine;
}

const toneClass: Record<LogLevelTone, string> = { warning: "text-warning", error: "text-destructive" };

// The text column wraps under itself, so a long line never runs back under its timestamp, and an indented line's wrap
// hangs under its own first character; a blank line keeps its row height. The level word takes its status hue.
export const LogLineRow = memo(({ line }: LogLineRowProps) => {
  const { indent, level, tone, rest } = logLineParts(line.text);
  return (
    <div
      data-stream={line.stream}
      className={cn(
        "relative grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 py-px pr-12 pl-4 font-mono text-xs leading-5",
        line.stream === "stderr" && "before:absolute before:inset-y-0 before:left-0 before:w-0.5 before:bg-destructive",
      )}
    >
      <span className="text-muted-foreground tabular-nums select-none">{formatContainerLogTime(line)}</span>
      <span className="[overflow-wrap:anywhere] whitespace-pre-wrap" style={indent > 0 ? { paddingLeft: `${indent}ch` } : undefined}>
        {level && <span className={tone ? toneClass[tone] : "text-muted-foreground"}>{level}</span>}
        {breakableParts(rest).map((part, i) => (
          <Fragment key={i}>
            {i > 0 && <wbr />}
            {part}
          </Fragment>
        ))}
        {!line.text && " "}
      </span>
    </div>
  );
});
