// Structured logs only (AGENTS.md hard rule 8): one JSON line per event on
// stdout, no console.log in the rest of the codebase.
export type Level = "info" | "warn" | "error";

export interface LogRecord {
  time: string;
  level: Level;
  message: string;
  attrs?: Record<string, unknown>;
}

export const QUEUE_LIMIT = 1000;
export const MAX_BATCH = 50;
export const MAX_BATCH_BYTES = 48 * 1024;
const MAX_RECORD_BYTES = 16 * 1024;
const MAX_CHARS = 2000;

// Same shapes as Go's redact.Tokens: dep_/ses_ tokens and any token= query value.
const BEARER = /(?:dep|ses)_[A-Za-z0-9_-]{43,}/g;
const QUERY = /([?&]token=)[^&#\s"'\\]+/g;
const PLACEHOLDER = "[redacted token]";

function clean(s: string): string {
  const t = s.replace(QUERY, `$1${PLACEHOLDER}`).replace(BEARER, PLACEHOLDER);
  return t.length > MAX_CHARS ? `${t.slice(0, MAX_CHARS)}…` : t;
}

function scrub(value: unknown): unknown {
  if (typeof value === "string") return clean(value);
  if (Array.isArray(value)) return value.map(scrub);
  if (value && typeof value === "object") return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, scrub(v)]));
  return value;
}

interface Queued {
  record: LogRecord;
  size: number;
}

// LogQueue keeps the newest QUEUE_LIMIT records the server has not taken. peek/ack mirror the Go queue: a batch that
// failed to send stays queued, and a drop count rides the next batch as one warning.
export class LogQueue {
  private items: Queued[] = [];
  private base = 0;
  private dropped = 0;

  add(record: LogRecord): void {
    const safe: LogRecord = { ...record, message: clean(record.message) };
    if (record.attrs) safe.attrs = scrub(record.attrs) as Record<string, unknown>;
    let size = JSON.stringify(safe).length;
    if (size > MAX_RECORD_BYTES) {
      delete safe.attrs;
      size = JSON.stringify(safe).length;
    }
    if (this.items.length >= QUEUE_LIMIT) {
      this.items.shift();
      this.base++;
      this.dropped++;
    }
    this.items.push({ record: safe, size });
  }

  peek(maxRecords = MAX_BATCH, maxBytes = MAX_BATCH_BYTES): { records: LogRecord[]; next: number; dropped: number } {
    const records: LogRecord[] = [];
    const dropped = this.dropped;
    let size = 0;
    if (dropped > 0) {
      records.push({ time: new Date().toISOString(), level: "warn", message: `dropped ${dropped} log lines` });
      size = 200;
    }
    let taken = 0;
    for (const it of this.items) {
      if (records.length >= maxRecords || (records.length > 0 && size + it.size > maxBytes)) break;
      records.push(it.record);
      size += it.size;
      taken++;
    }
    return { records, next: this.base + taken, dropped };
  }

  ack(next: number, dropped: number): void {
    const n = Math.min(Math.max(next - this.base, 0), this.items.length);
    this.items.splice(0, n);
    this.base += n;
    this.dropped = Math.max(this.dropped - dropped, 0);
  }

  get length(): number {
    return this.items.length;
  }
}

let queue: LogQueue | null = null;

// shipLogsTo queues every later log() record for the server; null stops queueing.
export function shipLogsTo(q: LogQueue | null): void {
  queue = q;
}

function write(level: Level, message: string, meta: Record<string, unknown>): void {
  const line = JSON.stringify({ ts: new Date().toISOString(), level, message, ...meta });
  if (level === "error") {
    console.error(line);
    return;
  }
  console.log(line);
}

// log is the host's own line: written locally and, once shipping is on, queued for the server.
export function log(level: Level, message: string, meta: Record<string, unknown> = {}): void {
  write(level, message, meta);
  queue?.add({ time: new Date().toISOString(), level, message, ...(Object.keys(meta).length > 0 && { attrs: meta }) });
}

// logScript writes a line an automation's own code printed. It stays local: a script's output belongs to its workspace
// and is kept with the run, not shipped as Nexul's.
export function logScript(level: Level, message: string, meta: Record<string, unknown> = {}): void {
  write(level, message, meta);
}
