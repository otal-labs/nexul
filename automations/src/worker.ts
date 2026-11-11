import { Worker } from "node:worker_threads";
import { log, logScript } from "./log.ts";

export interface SpawnOptions {
  automationId: string;
  automationName: string;
  code: string;
  url: string;
  token: string;
  timeoutMs: number;
  heartbeatTimeoutMs: number;
  memoryMb: number;
  // Answers the thread's request to resolve pool references, for this automation's workspace only.
  resolvePool: (entries: unknown) => Promise<Record<string, string>>;
}

export interface WorkerHandle {
  terminate(): Promise<void>;
}

// onStale fires when the worker ends without the supervisor asking for it
// (a missed heartbeat or an exit of its own), so its next poll respawns
// instead of assuming it's still up.
export interface WorkerFactory {
  spawn(opts: SpawnOptions, onStale: () => void): WorkerHandle;
}

// A compiled binary holds the worker entrypoint as worker-entry.js beside its bundled main; from source it is the .ts.
const WORKER_ENTRY = new URL(import.meta.url.endsWith(".ts") ? "./worker-entry.ts" : "./worker-entry.js", import.meta.url);

interface WorkerMessage {
  type: "log" | "heartbeat" | "resolve";
  id?: number;
  entries?: unknown;
  level?: "info" | "warn" | "error";
  message?: string;
  meta?: Record<string, unknown>;
}

export const threadWorkerFactory: WorkerFactory = {
  spawn(opts, onStale) {
    const worker = new Worker(WORKER_ENTRY, {
      workerData: {
        automationId: opts.automationId,
        code: opts.code,
        url: opts.url,
        token: opts.token,
        timeoutMs: opts.timeoutMs,
        heartbeatMs: Math.max(1000, Math.floor(opts.heartbeatTimeoutMs / 3)),
      },
      // Bun doesn't enforce resourceLimits as of 1.4 (research doc §1) —
      // kept so this starts working for free the moment the runtime does.
      resourceLimits: { maxOldGenerationSizeMb: opts.memoryMb },
    });

    let lastHeartbeat = Date.now();
    // Bun reports a terminated worker's exit as code 1, like Node, so an exit the host asked for is not a failure.
    let terminating = false;
    const terminate = async (): Promise<void> => {
      terminating = true;
      clearInterval(watchdog);
      await worker.terminate();
    };
    const watchdog = setInterval(
      () => {
        if (Date.now() - lastHeartbeat <= opts.heartbeatTimeoutMs) return;
        log("warn", "automation worker missed its heartbeat, terminating", { automationId: opts.automationId });
        void terminate().then(() => onStale());
      },
      Math.max(1000, Math.floor(opts.heartbeatTimeoutMs / 3)),
    );

    worker.on("message", (msg: WorkerMessage) => {
      if (msg.type === "heartbeat") {
        lastHeartbeat = Date.now();
        return;
      }
      if (msg.type === "resolve") {
        void opts
          .resolvePool(msg.entries)
          .catch(() => ({}))
          .then((values) => worker.postMessage({ type: "resolved", id: msg.id, values }));
        return;
      }
      logScript(msg.level ?? "info", msg.message ?? "", { automationId: opts.automationId, automationName: opts.automationName, ...msg.meta });
    });
    worker.on("error", (err: Error) => log("error", "automation worker error", { automationId: opts.automationId, error: err.message }));
    worker.on("exit", (code) => {
      clearInterval(watchdog);
      if (terminating) return;
      log("warn", "automation worker exited on its own", { automationId: opts.automationId, code });
      onStale();
    });

    return { terminate };
  },
};
