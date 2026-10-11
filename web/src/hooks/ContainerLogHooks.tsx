import { useEffect, useState } from "react";

import { ContainerLogStream } from "@/api/logStream";
import type { ContainerLogs } from "@/models/ContainerLog";
import { useSessionStore } from "@/stores/sessionStore";
import { appendLogLines } from "@/utils/ContainerLogUtility";

const initial: ContainerLogs = { lines: [], status: "connecting", reason: undefined };

// Follows the tail at path while mounted; unmounting or a new path closes the socket, and the caller keys on the path.
export const useContainerLogs = (path: string): ContainerLogs => {
  const [logs, setLogs] = useState(initial);
  const token = useSessionStore((s) => s.token);

  useEffect(() => {
    if (!token) return;
    const stream = new ContainerLogStream(path, token, {
      onLines: (incoming) => setLogs((prev) => ({ ...prev, lines: appendLogLines(prev.lines, incoming) })),
      onStatus: (status, reason) => setLogs((prev) => ({ ...prev, status, reason })),
    });
    stream.open();
    return () => stream.close();
  }, [path, token]);

  return logs;
};
