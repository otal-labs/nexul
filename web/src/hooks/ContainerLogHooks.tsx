import { useEffect, useState } from "react";

import { createFrameBatch } from "@nexul/client-core/frameBatch";

import { ContainerLogStream } from "@/api/logStream";
import { MAX_LOG_LINES, type ContainerLogs, type ContainerLogWireLine } from "@/models/ContainerLog";
import { useSessionStore } from "@/stores/sessionStore";
import { useIsViewingAs } from "@/stores/viewAsStore";
import { appendLogLines } from "@/utils/ContainerLogUtility";

const initial: ContainerLogs = { lines: [], status: "connecting", reason: undefined };
const viewingLogs: ContainerLogs = { lines: [], status: "forbidden", reason: "Live logs are off while viewing as another member." };

// Follows the tail at path while mounted; unmounting or a new path closes the socket, and the caller keys on the path.
export const useContainerLogs = (path: string): ContainerLogs => {
  const [logs, setLogs] = useState(initial);
  const token = useSessionStore((s) => s.token);
  const viewing = useIsViewingAs();

  useEffect(() => {
    if (!token || viewing) return;
    const batch = createFrameBatch<ContainerLogWireLine>({
      deliver: (incoming) => setLogs((prev) => ({ ...prev, lines: appendLogLines(prev.lines, incoming) })),
      limit: MAX_LOG_LINES,
      requestFrame: requestAnimationFrame,
      cancelFrame: cancelAnimationFrame,
    });
    const stream = new ContainerLogStream(path, token, {
      onLines: batch.push,
      onStatus: (status, reason) => {
        batch.flush();
        setLogs((prev) => ({ ...prev, status, reason }));
      },
    });
    stream.open();
    return () => {
      stream.close();
      batch.cancel();
    };
  }, [path, token, viewing]);

  return viewing ? viewingLogs : logs;
};
