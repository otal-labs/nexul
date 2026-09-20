import { Worker } from "node:worker_threads";
import { log } from "./log.ts";

export interface SpawnOptions {
  automationId: string;
  automationName: string;
  code: string;
  url: string;
  token: string;
  timeoutMs: number;
  heartbeatTimeoutMs: number;
  memoryMb: number;
}

export interface WorkerHandle {
  terminate(): Promise<void>;
}

// onStale fires when the watchdog kills the worker itself (missed
// heartbeat) rather than the supervisor asking for it — the supervisor
// needs to know so its next poll respawns instead of assuming it's still up.
export interface WorkerFactory {
  spawn(opts: SpawnOptions, onStale: () => void): WorkerHandle;
}

const WORKER_ENTRY = new URL("./worker-entry.ts", import.meta.url);

interface WorkerMessage {
  type: "log" | "heartbeat";
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
    const watchdog = setInterval(
      () => {
        if (Date.now() - lastHeartbeat <= opts.heartbeatTimeoutMs) return;
        log("warn", "automation worker missed its heartbeat, terminating", { automationId: opts.automationId });
        clearInterval(watchdog);
        void worker.terminate().then(() => onStale());
      },
      Math.max(1000, Math.floor(opts.heartbeatTimeoutMs / 3)),
    );

    worker.on("message", (msg: WorkerMessage) => {
      if (msg.type === "heartbeat") {
        lastHeartbeat = Date.now();
        return;
      }
      log(msg.level ?? "info", msg.message ?? "", { automationId: opts.automationId, automationName: opts.automationName, ...msg.meta });
    });
    worker.on("error", (err: Error) => log("error", "automation worker error", { automationId: opts.automationId, error: err.message }));
    worker.on("exit", (code) => {
      clearInterval(watchdog);
      if (code !== 0) log("warn", "automation worker exited non-zero", { automationId: opts.automationId, code });
    });

    return {
      async terminate() {
        clearInterval(watchdog);
        await worker.terminate();
      },
    };
  },
};
