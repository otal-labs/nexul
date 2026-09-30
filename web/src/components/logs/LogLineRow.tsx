import { memo } from "react";

import { cn } from "@/lib/utils";
import type { ContainerLogLine } from "@/models/ContainerLog";
import { formatContainerLogTime } from "@/utils/ContainerLogUtility";

interface LogLineRowProps {
  line: ContainerLogLine;
}

// The text column wraps under itself, so a long line never runs back under its timestamp; a blank line keeps its row height.
export const LogLineRow = memo(({ line }: LogLineRowProps) => (
  <div
    data-stream={line.stream}
    className={cn(
      "relative grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 py-px pr-12 pl-4 font-mono text-xs leading-5",
      line.stream === "stderr" && "before:absolute before:inset-y-0 before:left-0 before:w-0.5 before:bg-destructive",
    )}
  >
    <span className="text-muted-foreground tabular-nums select-none">{formatContainerLogTime(line)}</span>
    <span className="[overflow-wrap:anywhere] whitespace-pre-wrap">{line.text || " "}</span>
  </div>
));
