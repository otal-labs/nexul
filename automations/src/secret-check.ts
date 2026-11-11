import { log } from "./log.ts";
import type { ResolveResult } from "./resolver.ts";

export const MIN_CHECK_INTERVAL_MS = 5 * 60_000;
export const DEFAULT_CHECK_INTERVAL_MS = 60 * 60_000;

const UNIT_MS = { s: 1000, m: 60_000, h: 3_600_000 } as const;

// Reads NEXUL_SECRET_CHECK_INTERVAL like the runner does: a duration such as 30m or 1h, or "off"; 0 means off. Every
// check reads the secrets manager, which limits reads, so anything under five minutes is refused.
export function parseCheckInterval(raw: string | undefined): number {
  const value = raw?.trim();
  if (!value) return DEFAULT_CHECK_INTERVAL_MS;
  if (value.toLowerCase() === "off") return 0;
  const parts = [...value.matchAll(/(\d+)([smh])/g)];
  if (parts.length === 0 || parts.map((p) => p[0]).join("") !== value) {
    throw new Error(`NEXUL_SECRET_CHECK_INTERVAL "${value}" is not a duration like 30m or 1h, or "off"`);
  }
  const ms = parts.reduce((sum, [, n, unit]) => sum + Number(n) * UNIT_MS[unit as keyof typeof UNIT_MS], 0);
  if (ms < MIN_CHECK_INTERVAL_MS) {
    throw new Error(`NEXUL_SECRET_CHECK_INTERVAL "${value}" is under the 5m minimum: each check reads the secrets manager, which limits reads per hour and per day`);
  }
  return ms;
}

// after runs fn once in ms and returns what cancels it.
export interface Clock {
  after(ms: number, fn: () => void): () => void;
}

export const realClock: Clock = {
  after(ms, fn) {
    const timer = setTimeout(fn, ms);
    timer.unref();
    return () => clearTimeout(timer);
  },
};

export interface SecretReader {
  resolve(workspaceId: string, refs: string[]): Promise<Map<string, ResolveResult>>;
}

export interface Checkable {
  checkSecrets(reader: SecretReader): Promise<void>;
}

// Runs the supervisor's check every intervalMs, the next one scheduled once the last ends; zero makes no reads at all.
export class SecretCheck {
  private cancel: (() => void) | undefined;
  private stopped = false;

  constructor(
    private readonly target: Checkable,
    private readonly reader: SecretReader,
    private readonly intervalMs: number,
    private readonly clock: Clock = realClock,
  ) {}

  start(): void {
    if (this.intervalMs === 0) return;
    this.cancel = this.clock.after(this.intervalMs, () => void this.tick());
  }

  stop(): void {
    this.stopped = true;
    this.cancel?.();
  }

  private async tick(): Promise<void> {
    try {
      await this.target.checkSecrets(this.reader);
    } catch (err) {
      log("warn", "secret check failed; it is tried again next time", { error: err instanceof Error ? err.name : "unknown" });
    }
    if (!this.stopped) this.start();
  }
}
