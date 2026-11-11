import { useMemo } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { cn } from "@/lib/utils";
import { automationLogLine } from "@/utils/AutomationRunUtility";

interface AutomationRunLogViewProps {
  logs: string;
}

const levelClass: Record<string, string> = { warn: "text-warning", error: "text-destructive" };

// One row per line: the time, the level (blank for info) and the message, the other fields muted under it; capped at 1MB server-side.
export const AutomationRunLogView = ({ logs }: AutomationRunLogViewProps) => {
  const lines = useMemo(() => (logs.trim() === "" ? [] : logs.trimEnd().split("\n").map(automationLogLine)), [logs]);

  return (
    <div className="terminal-window">
      <div className="terminal-window__bar">
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__title">Logs</span>
      </div>
      {lines.length === 0 && (
        <div className="terminal-window__body">
          <EmptyRow>No logs captured</EmptyRow>
        </div>
      )}
      {lines.length > 0 && (
        <div className="terminal-window__body space-y-1">
          {lines.map((line, index) => (
            <p key={index} className="grid grid-cols-[auto_5ch_minmax(0,1fr)] gap-x-3">
              <span className="text-muted-foreground tabular-nums">{line.time}</span>
              <span className={cn(line.level && levelClass[line.level])}>{line.level !== "info" && line.level}</span>
              <span className="break-words">
                {line.message}
                {line.fields && <span className="block text-muted-foreground">{line.fields}</span>}
              </span>
            </p>
          ))}
        </div>
      )}
    </div>
  );
};
