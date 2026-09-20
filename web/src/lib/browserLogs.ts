import { api } from "@/api/client";
import { useSessionStore } from "@/stores/sessionStore";

type Level = "warn" | "error";

interface BrowserLogRecord {
  level: Level;
  message: string;
  url: string;
  attrs?: Record<string, unknown>;
}

const FLUSH_DELAY_MS = 2_000;
const MAX_QUEUE = 200;
const MAX_BATCH = 50;

const queue: BrowserLogRecord[] = [];
let timer: number | undefined;

const describe = (value: unknown): string => {
  if (value instanceof Error) return `${value.name}: ${value.message}`;
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
};

const stackOf = (value: unknown): string | undefined =>
  value instanceof Error ? value.stack : undefined;

const flush = () => {
  timer = undefined;
  // Before login there is no session to attach and a 401 would bounce the user; the queue waits for the next tick.
  if (!useSessionStore.getState().token) return;
  const records = queue.splice(0, MAX_BATCH);
  if (records.length === 0) return;
  // ponytail: dropped on failure, no retry — a log relay that retries can amplify the outage it reports.
  void api.post("/api/logs", { records }).catch(() => undefined);
  if (queue.length > 0) schedule();
};

const schedule = () => {
  timer ??= window.setTimeout(flush, FLUSH_DELAY_MS);
};

const enqueue = (level: Level, message: string, attrs?: Record<string, unknown>) => {
  if (queue.length >= MAX_QUEUE) queue.shift();
  queue.push({ level, message, url: window.location.pathname, ...(attrs && { attrs }) });
  schedule();
};

// Captures console.error/warn, uncaught errors, and unhandled rejections and relays them to POST /api/logs,
// so the browser shows up in OpenObserve next to the server. Call once at boot.
export const installBrowserLogs = () => {
  window.addEventListener("error", (event) =>
    enqueue("error", event.message, { stack: stackOf(event.error), file: event.filename, line: event.lineno }),
  );
  window.addEventListener("unhandledrejection", (event) =>
    enqueue("error", `unhandled rejection: ${describe(event.reason)}`, { stack: stackOf(event.reason) }),
  );
  for (const level of ["error", "warn"] as const) {
    const original = console[level];
    console[level] = (...args: unknown[]) => {
      original(...args);
      const stack = args.map(stackOf).find(Boolean);
      enqueue(level, args.map(describe).join(" "), stack ? { stack } : undefined);
    };
  }
};
