import { requires } from "@nexul/client-core/requires";

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
  void requires("none", api.post("/api/logs", { records })).catch(() => undefined);
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

const STALL_MS = 2_000;
const MAX_SCRIPTS = 3;

// The fields of a long animation frame entry read here; TypeScript's DOM lib does not declare the type yet.
interface FrameScript {
  invoker: string;
  sourceURL: string;
  sourceFunctionName: string;
  sourceCharPosition: number;
  duration: number;
  forcedStyleAndLayoutDuration: number;
}
interface LongFrame {
  startTime: number;
  duration: number;
  blockingDuration: number;
  styleAndLayoutStart: number;
  scripts: FrameScript[];
}

const scriptLine = (s: FrameScript) =>
  `${Math.round(s.duration)}ms ${s.invoker} → ${s.sourceFunctionName || "(anonymous)"} ${s.sourceURL}:${s.sourceCharPosition} (forced layout ${Math.round(s.forcedStyleAndLayoutDuration)}ms)`;

// Reports, once the tab recovers, a frame that held the main thread for seconds, with the scripts that ran in it.
const reportStall = (frame: LongFrame) => {
  const end = frame.startTime + frame.duration;
  const scripts = [...frame.scripts].sort((a, b) => b.duration - a.duration).slice(0, MAX_SCRIPTS);
  enqueue("warn", `main thread stalled for ${Math.round(frame.duration)}ms`, {
    duration_ms: Math.round(frame.duration),
    blocking_ms: Math.round(frame.blockingDuration),
    layout_ms: frame.styleAndLayoutStart > 0 ? Math.round(end - frame.styleAndLayoutStart) : 0,
    script_ms: Math.round(frame.scripts.reduce((sum, s) => sum + s.duration, 0)),
    ...Object.fromEntries(scripts.map((s, i) => [`script_${i + 1}`, scriptLine(s)])),
  });
};

const observeStalls = () => {
  if (!PerformanceObserver.supportedEntryTypes?.includes("long-animation-frame")) return;
  new PerformanceObserver((list) => {
    for (const entry of list.getEntries() as unknown as LongFrame[]) {
      if (entry.duration >= STALL_MS) reportStall(entry);
    }
  }).observe({ type: "long-animation-frame", buffered: true });
};

// Captures console.error/warn, uncaught errors, unhandled rejections, and multi-second main-thread stalls and relays
// them to POST /api/logs, so the browser shows up in OpenObserve next to the server. Call once at boot.
export const installBrowserLogs = () => {
  if (typeof PerformanceObserver !== "undefined") observeStalls();
  window.addEventListener("error", (event) =>
    enqueue("error", event.message, { stack: stackOf(event.error), file: event.filename, line: event.lineno }),
  );
  window.addEventListener("unhandledrejection", (event) =>
    enqueue("error", `unhandled rejection: ${describe(event.reason)}`, { stack: stackOf(event.reason) }),
  );
  /* eslint-disable no-console -- this hook wraps console.error/warn to relay them */
  for (const level of ["error", "warn"] as const) {
    const original = console[level];
    console[level] = (...args: unknown[]) => {
      original(...args);
      const stack = args.map(stackOf).find(Boolean);
      enqueue(level, args.map(describe).join(" "), stack ? { stack } : undefined);
    };
  }
  /* eslint-enable no-console */
};
