import { useMemo, useState } from "react";
import { toast } from "sonner";

import { LogBlock } from "@/components/logs/LogBlock";
import { LogToolbar } from "@/components/logs/LogToolbar";
import { MessageScrollerProvider } from "@/components/ui/message-scroller";
import { useContainerLogs } from "@/hooks/ContainerLogHooks";
import type { ContainerLogLine, LogFilter } from "@/models/ContainerLog";
import { containerLogToText, emptyLogMessage, filterLogLines, saveTextFile } from "@/utils/ContainerLogUtility";

interface LogsViewProps {
  stackId: string;
  service: string;
}

// Pause freezes what is on screen while the stream keeps buffering; resuming shows everything that arrived.
export const LogsView = ({ stackId, service }: LogsViewProps) => {
  const logs = useContainerLogs(stackId, service);
  const [frozen, setFrozen] = useState<ContainerLogLine[] | null>(null);
  const [filter, setFilter] = useState<LogFilter>("all");

  const shown = useMemo(() => filterLogLines(frozen ?? logs.lines, filter), [frozen, logs.lines, filter]);

  const copy = async () => {
    await navigator.clipboard.writeText(containerLogToText(shown));
    toast.success("Log copied");
  };

  return (
    <MessageScrollerProvider autoScroll defaultScrollPosition="end">
      <div className="space-y-3">
        <LogToolbar
          filter={filter}
          onFilterChange={setFilter}
          status={logs.status}
          reason={logs.reason}
          paused={frozen !== null}
          onPausedChange={(paused) => setFrozen(paused ? logs.lines : null)}
          onCopy={() => void copy()}
          onDownload={() => saveTextFile(`${service}.log`, containerLogToText(shown))}
          empty={shown.length === 0}
        />
        <LogBlock lines={shown} empty={emptyLogMessage(logs, filter)} onCopy={() => void copy()} />
      </div>
    </MessageScrollerProvider>
  );
};
