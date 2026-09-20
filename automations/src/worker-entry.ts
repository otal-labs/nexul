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
import { DialinClient } from "@nexul/sdk/client";
import type { Automation } from "@nexul/sdk/automation";

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

async function main(): Promise<void> {
  const { automationId, code, url, token, timeoutMs, heartbeatMs } = workerData as WorkerData;
  const automation = await loadAutomation(automationId, code);

  const heartbeat = setInterval(() => parentPort?.postMessage({ type: "heartbeat" }), heartbeatMs);
  heartbeat.unref();

  const client = new DialinClient(automation, {
    url,
    token,
    timeoutMs,
    logger: (level, message, meta) => parentPort?.postMessage({ type: "log", level, message, meta }),
  });
  await client.run();
}

main().catch((err) => {
  parentPort?.postMessage({ type: "log", level: "error", message: "worker crashed", meta: { error: err instanceof Error ? err.message : String(err) } });
  process.exit(1);
});
