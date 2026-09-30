import { useFocusEffect } from "expo-router";
import { useCallback, useState } from "react";

import { api } from "@/api/client";
import { errorMessage, isNotFound } from "@/api/errors";
import { buildLogsURL, parseLogFrame } from "@/api/logs";
import type { ContainerLogEntry, ContainerLogLine } from "@/models/Stack";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

export const MAX_LOG_LINES = 2000;
const FIRST_TAIL = 200;
const RECONNECT_TAIL = 20;
const BASE_BACKOFF_MS = 500;
const MAX_BACKOFF_MS = 30_000;
const JITTER_MS = 250;

// ended: the container stopped writing. forbidden: the server refused this viewer, so retrying is pointless.
export type LogStatus = "connecting" | "live" | "offline" | "ended" | "forbidden";

export interface ContainerLogs {
  lines: ContainerLogEntry[];
  status: LogStatus;
  reason: string | undefined;
}

const initial: ContainerLogs = { lines: [], status: "connecting", reason: undefined };

// Follows one service's tail while its screen is focused; blur or unmount closes the socket, which stops the stream on the host.
export const useContainerLogs = (stackId: string, service: string, enabled: boolean): ContainerLogs => {
  const [logs, setLogs] = useState(initial);
  const host = useSessionStore((s) => s.host);

  useFocusEffect(
    useCallback(() => {
      if (!enabled || !host) return;
      setLogs(initial);
      let socket: WebSocket | null = null;
      let timer: ReturnType<typeof setTimeout> | null = null;
      let attempt = 0;
      let stopped = false;
      let newest = "";

      const append = (incoming: ContainerLogLine[], after: string) =>
        setLogs((prev) => {
          const fresh = after ? incoming.filter((l) => l.ts > after) : incoming;
          if (fresh.length === 0) return prev;
          let id = (prev.lines.at(-1)?.id ?? -1) + 1;
          const added = fresh.map((l) => ({ ...l, id: id++ }));
          return { ...prev, lines: [...prev.lines, ...added].slice(-MAX_LOG_LINES) };
        });

      const retry = (reason: string | undefined) => {
        if (stopped) return;
        setLogs((prev) => ({ ...prev, status: "offline", reason }));
        const delay = Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** attempt) + Math.random() * JITTER_MS;
        attempt += 1;
        timer = setTimeout(connect, delay);
      };

      // A refused handshake reaches the socket as a bare close, so the snapshot route says why.
      const explainFailure = () =>
        api.get(`/api/stacks/${stackId}/services/${encodeURIComponent(service)}/logs?tail=1`).then(
          () => retry(undefined),
          (error: unknown) => {
            if (stopped) return;
            if (isNotFound(error)) {
              setLogs((prev) => ({ ...prev, status: "forbidden" }));
              return;
            }
            retry(errorMessage(error));
          },
        );

      const connect = () => {
        const token = readSessionToken();
        if (stopped || !token) return;
        const after = newest;
        let opened = false;
        const ws = new WebSocket(buildLogsURL(host, token, stackId, service, after ? RECONNECT_TAIL : FIRST_TAIL));
        socket = ws;
        ws.onopen = () => {
          opened = true;
          attempt = 0;
          setLogs((prev) => ({ ...prev, status: "live", reason: undefined }));
        };
        ws.onmessage = (ev) => {
          try {
            const lines = parseLogFrame(String(ev.data));
            lines.forEach((l) => {
              if (l.ts > newest) newest = l.ts;
            });
            append(lines, after);
          } catch {
            console.warn("logs dropped malformed frame");
          }
        };
        ws.onclose = (ev) => {
          socket = null;
          if (stopped) return;
          if (!opened) {
            void explainFailure();
            return;
          }
          if (ev.code === 1000) {
            setLogs((prev) => ({ ...prev, status: "ended" }));
            return;
          }
          retry(ev.reason || undefined);
        };
      };

      connect();
      return () => {
        stopped = true;
        if (timer) clearTimeout(timer);
        if (socket) {
          socket.onopen = null;
          socket.onmessage = null;
          socket.onclose = null;
          socket.close();
        }
      };
    }, [enabled, host, stackId, service]),
  );

  return logs;
};
