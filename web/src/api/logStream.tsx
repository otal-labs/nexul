import type { AxiosError } from "axios";

import { closeSocket, type LiveSocket } from "@/api/ws";
import { api, errorMessage } from "@/api/client";
import { buildLogsURL } from "@/lib/live";
import type { ContainerLogWireLine, LogStatus } from "@/models/ContainerLog";

const FIRST_TAIL = 200;
const RECONNECT_TAIL = 20;
const BASE_BACKOFF_MS = 500;
const MAX_BACKOFF_MS = 30_000;
const JITTER_MS = 250;

export interface ContainerLogStreamHandlers {
  onLines: (lines: ContainerLogWireLine[]) => void;
  onStatus: (status: LogStatus, reason?: string) => void;
}

// One viewer's tail of one service: the socket's lifetime is the stream's, so close() stops it on the host.
export class ContainerLogStream {
  private socket: WebSocket | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private attempt = 0;
  private closed = false;
  private newest = "";

  constructor(
    private readonly stackId: string,
    private readonly service: string,
    private readonly token: string,
    private readonly handlers: ContainerLogStreamHandlers,
  ) {}

  open() {
    if (this.closed) return;
    // A reconnect replays a small tail; lines at or before the newest one already seen are the overlap.
    const seen = this.newest;
    let opened = false;
    const ws = new WebSocket(buildLogsURL(this.token, this.stackId, this.service, seen ? RECONNECT_TAIL : FIRST_TAIL));
    this.socket = ws;
    ws.onopen = () => {
      opened = true;
      this.attempt = 0;
      this.handlers.onStatus("live");
    };
    ws.onmessage = (ev) => this.receive(String(ev.data), seen);
    ws.onclose = (ev) => {
      this.socket = null;
      if (!opened) {
        void this.explainRefusal();
        return;
      }
      if (ev.code === 1000) {
        this.handlers.onStatus("ended");
        return;
      }
      this.retry(ev.reason || undefined);
    };
  }

  close() {
    this.closed = true;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    const socket = this.socket;
    this.socket = null;
    if (!socket) return;
    // Detach first, or the torn-down socket's close event schedules a reconnect.
    socket.onmessage = null;
    socket.onclose = null;
    closeSocket(socket as unknown as LiveSocket);
  }

  private receive(raw: string, seen: string) {
    try {
      const data = JSON.parse(raw) as { lines?: unknown };
      if (!Array.isArray(data.lines)) throw new Error("frame has no lines");
      const lines = data.lines as ContainerLogWireLine[];
      for (const l of lines) if (l.ts > this.newest) this.newest = l.ts;
      const fresh = seen ? lines.filter((l) => l.ts > seen) : lines;
      if (fresh.length > 0) this.handlers.onLines(fresh);
    } catch {
      console.warn("logs dropped malformed frame", { stackId: this.stackId, service: this.service });
    }
  }

  // A refused handshake reaches the socket as a bare close, so the snapshot route says why.
  private async explainRefusal() {
    try {
      await api.get(`/api/stacks/${this.stackId}/services/${encodeURIComponent(this.service)}/logs?tail=1`);
    } catch (error) {
      if (this.closed) return;
      const status = (error as AxiosError).response?.status;
      if (status === 403 || status === 404) {
        this.handlers.onStatus("forbidden");
        return;
      }
      this.retry(errorMessage(error));
      return;
    }
    this.retry(undefined);
  }

  private retry(reason: string | undefined) {
    if (this.closed) return;
    this.handlers.onStatus("offline", reason);
    const delay = Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** this.attempt) + Math.random() * JITTER_MS;
    this.attempt += 1;
    this.timer = setTimeout(() => this.open(), delay);
  }
}
