import { ApiClient } from "./api-client.ts";
import { buildCtx } from "./context.ts";
import { defaultConfigValues, toJsonSchema, type ConfigSchema, type ConfigValues } from "./config-schema.ts";
import type { Automation } from "./define-automation.ts";
import type { EventPayloads, Topic } from "./events.generated.ts";
import { announceFrame, runCrashedFrame, runFinishedFrame, runLogFrame, runStartedFrame, type Frame } from "./protocol.ts";

const DEFAULT_TIMEOUT_MS = 30_000;
const MAX_BACKOFF_MS = 30_000;

export type LogFn = (level: "info" | "warn" | "error", message: string, meta?: Record<string, unknown>) => void;

export interface DialinOptions {
  url: string;
  token: string;
  timeoutMs?: number;
  logger?: LogFn;
}

class TimeoutError extends Error {}

function toWsUrl(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, "").replace(/^http/, "ws") + "/ws/automations";
}

function withTimeout<T>(p: Promise<T>, ms: number): Promise<T> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new TimeoutError(`handler timed out after ${ms}ms`)), ms);
    p.then(
      (v) => {
        clearTimeout(timer);
        resolve(v);
      },
      (err) => {
        clearTimeout(timer);
        reject(err);
      },
    );
  });
}

// withCapturedConsole redirects console output to `log` for the duration of
// one handler call, so a handler that reaches for plain console.log (as most
// will, out of habit) still shows up in the run's captured logs
// without every automation author having to learn ctx.log first. Safe only
// because delivery is strictly sequential (ADR 0046): exactly one handler
// runs at a time, so there's never a second call racing this global swap.
async function withCapturedConsole<T>(fn: () => Promise<T>, log: (message: string) => void): Promise<T> {
  const original = { log: console.log, error: console.error, warn: console.warn, info: console.info };
  const capture = (...args: unknown[]) => log(args.map((a) => (typeof a === "string" ? a : JSON.stringify(a))).join(" "));
  console.log = capture;
  console.error = capture;
  console.warn = capture;
  console.info = capture;
  try {
    return await fn();
  } finally {
    Object.assign(console, original);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// DialinClient is the SDK's implementation of the server's dial-in protocol
// (internal/automations/dialin.go): connect, announce first, receive
// config/secrets on hello, then process events one at a time, reporting
// each run back before the server sends the next (ADR 0046).
export class DialinClient<S extends ConfigSchema> {
  private readonly api: ApiClient;
  private readonly log: LogFn;
  private config: ConfigValues<S>;
  private secrets: Record<string, string> = {};
  private ws: WebSocket | undefined;
  private stopped = false;

  constructor(
    private readonly automation: Automation<S>,
    private readonly opts: DialinOptions,
  ) {
    this.api = new ApiClient({ baseUrl: opts.url, token: opts.token });
    this.log = opts.logger ?? ((level, message, meta) => console.error(JSON.stringify({ level, message, ...meta })));
    this.config = defaultConfigValues(automation.configSchema);
  }

  // run connects and reconnects with exponential backoff until stop() is
  // called — the SDK's own resilience, since nothing supervises a
  // bring-your-own-server automation the way the automations host would.
  async run(): Promise<void> {
    let backoffMs = 1000;
    while (!this.stopped) {
      try {
        await this.connectOnce();
        backoffMs = 1000;
      } catch (err) {
        this.log("error", "automation dial-in connection failed", { error: String(err) });
      }
      if (this.stopped) return;
      await sleep(backoffMs);
      backoffMs = Math.min(backoffMs * 2, MAX_BACKOFF_MS);
    }
  }

  stop(): void {
    this.stopped = true;
    this.ws?.close();
  }

  private connectOnce(): Promise<void> {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(`${toWsUrl(this.opts.url)}?token=${encodeURIComponent(this.opts.token)}`);
      this.ws = ws;
      let settled = false;

      ws.addEventListener("open", () => {
        ws.send(
          JSON.stringify(
            announceFrame(this.automation.name, this.automation.description, this.automation.subscriptions, toJsonSchema(this.automation.configSchema)),
          ),
        );
        this.log("info", "automation connected", { name: this.automation.name });
      });
      ws.addEventListener("message", (ev) => {
        void this.handleFrame(JSON.parse(ev.data as string) as Frame, ws).catch((err) =>
          this.log("error", "frame handling failed", { error: String(err) }),
        );
      });
      ws.addEventListener("close", () => {
        if (settled) return;
        settled = true;
        resolve();
      });
      ws.addEventListener("error", () => {
        if (settled) return;
        settled = true;
        reject(new Error("automation websocket error"));
      });
    });
  }

  private async handleFrame(frame: Frame, ws: WebSocket): Promise<void> {
    if (frame.type === "hello") {
      this.config = (frame.config_values as ConfigValues<S>) ?? defaultConfigValues(this.automation.configSchema);
      this.secrets = frame.secrets ?? {};
      return;
    }
    if (frame.type === "event") {
      await this.runHandler(frame, ws);
    }
  }

  private async runHandler(frame: Frame, ws: WebSocket): Promise<void> {
    const runId = frame.run_id;
    const topic = frame.topic;
    if (!runId || !topic) return;
    ws.send(JSON.stringify(runStartedFrame(runId)));

    const handler = this.automation.getHandler(topic);
    if (!handler) {
      ws.send(JSON.stringify(runCrashedFrame(runId, `no handler registered for topic ${topic}`)));
      return;
    }

    const emitLog = (message: string, meta?: Record<string, unknown>) => {
      const line = JSON.stringify({ ts: new Date().toISOString(), run_id: runId, message, ...meta });
      ws.send(JSON.stringify(runLogFrame(runId, `${line}\n`)));
    };
    const ctx = buildCtx(this.api, this.config, this.secrets, emitLog);
    const timeoutMs = this.opts.timeoutMs ?? DEFAULT_TIMEOUT_MS;

    try {
      const payload = frame.payload as EventPayloads[Topic];
      const success = await withTimeout(withCapturedConsole(() => handler(payload, ctx), emitLog), timeoutMs);
      ws.send(JSON.stringify(runFinishedFrame(runId, success === true ? "success" : "failure")));
    } catch (err) {
      if (err instanceof TimeoutError) {
        emitLog(err.message);
        ws.send(JSON.stringify(runFinishedFrame(runId, "failure")));
        return;
      }
      ws.send(JSON.stringify(runCrashedFrame(runId, err instanceof Error ? err.message : String(err))));
    }
  }
}
