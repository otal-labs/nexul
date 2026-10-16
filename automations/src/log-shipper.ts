import type { LogQueue, LogRecord } from "./log.ts";

export const FLUSH_MS = 2000;

export type PostLogs = (records: LogRecord[]) => Promise<void>;

export function httpPostLogs(serverUrl: string, credential: string): PostLogs {
  return async (records) => {
    const url = `${serverUrl}/api/automation-hosts/self/logs`;
    const res = await fetch(url, {
      method: "POST",
      headers: { Authorization: `Bearer ${credential}`, "Content-Type": "application/json" },
      body: JSON.stringify({ records }),
      signal: AbortSignal.timeout(10_000),
    });
    if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`);
  };
}

// LogShipper sends the queued records to the server every FLUSH_MS. A failed batch stays queued for the next try, and
// a failure is never logged: it would only queue another line for the same failing request.
export class LogShipper {
  private timer: ReturnType<typeof setInterval> | undefined;
  private flushing: Promise<void> | null = null;

  constructor(
    private readonly queue: LogQueue,
    private readonly post: PostLogs,
  ) {}

  start(everyMs = FLUSH_MS): void {
    this.timer = setInterval(() => void this.flush(), everyMs);
  }

  stop(): void {
    clearInterval(this.timer);
  }

  // flush sends batches until the queue is empty or one fails; a flush already running is joined, not doubled.
  flush(): Promise<void> {
    this.flushing ??= this.drain().finally(() => {
      this.flushing = null;
    });
    return this.flushing;
  }

  private async drain(): Promise<void> {
    for (;;) {
      const { records, next, dropped } = this.queue.peek();
      if (records.length === 0) return;
      try {
        await this.post(records);
      } catch {
        return;
      }
      this.queue.ack(next, dropped);
    }
  }
}
