// Runs inside the worker_thread (one per automation). Loads the
// version's bundled code (the same esm-bundle shape `nexul push`
// produces, see sdk/bin/cli.ts's cmdPush) from a temp file so plain ESM
// import() resolves it — a fresh worker means a fresh module graph every
// time, so there is no stale-cache problem to solve (research doc §3).
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { parentPort, workerData } from "node:worker_threads";
import { DialinClient } from "@otal-labs/nexul-sdk/client";
import { parseSecretReference } from "./secret-reference.ts";
import type { Automation } from "@otal-labs/nexul-sdk/automation";

export interface WorkerData {
  automationId: string;
  code: string;
  url: string;
  token: string;
  timeoutMs: number;
  heartbeatMs: number;
}

async function loadAutomation(automationId: string, code: string): Promise<Automation> {
  const dir = mkdtempSync(join(tmpdir(), `automations-${automationId}-`));
  const file = join(dir, "bundle.mjs");
  writeFileSync(file, code);
  const mod = (await import(pathToFileURL(file).href)) as { default?: Automation };
  if (!mod.default) throw new Error("automation bundle has no default export");
  return mod.default;
}

let nextRequest = 0;
const waiting = new Map<number, (values: Record<string, string>) => void>();
parentPort?.on("message", (msg: { type?: string; id?: number; values?: Record<string, string> }) => {
  if (msg.type !== "resolved" || msg.id === undefined) return;
  waiting.get(msg.id)?.(msg.values ?? {});
  waiting.delete(msg.id);
});

// Asks the supervisor, which alone holds the secrets manager tokens, to resolve references.
function askSupervisor(entries: Record<string, string>): Promise<Record<string, string>> {
  const id = nextRequest++;
  return new Promise((resolve) => {
    waiting.set(id, resolve);
    parentPort?.postMessage({ type: "resolve", id, entries });
  });
}

const isReference = (value: string): boolean => parseSecretReference(value).kind === "reference";

// Swaps each reference in the pool for its value; an entry that did not resolve is left out.
async function resolvePool(pool: Record<string, string>): Promise<Record<string, string>> {
  const refs = Object.entries(pool).filter(([, value]) => isReference(value));
  if (refs.length === 0) return pool;
  const values = await askSupervisor(Object.fromEntries(refs));
  const out: Record<string, string> = {};
  for (const [name, value] of Object.entries(pool)) {
    if (!isReference(value)) {
      out[name] = value;
      continue;
    }
    const resolved = values[name];
    if (resolved !== undefined) out[name] = resolved;
  }
  return out;
}

async function main(): Promise<void> {
  const { automationId, code, url, token, timeoutMs, heartbeatMs } = workerData as WorkerData;
  const automation = await loadAutomation(automationId, code);

  const heartbeat = setInterval(() => parentPort?.postMessage({ type: "heartbeat" }), heartbeatMs);
  heartbeat.unref();

  const client = new DialinClient(automation, {
    url,
    token,
    timeoutMs,
    resolveSecrets: resolvePool,
    onRunState: (running) => parentPort?.postMessage({ type: "run", running }),
    logger: (level, message, meta) => parentPort?.postMessage({ type: "log", level, message, meta }),
  });
  await client.run();
}

main().catch((err) => {
  parentPort?.postMessage({ type: "log", level: "error", message: "worker crashed", meta: { error: err instanceof Error ? err.message : String(err) } });
  process.exit(1);
});
