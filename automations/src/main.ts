import { httpAutomationsApi } from "./automations-api.ts";
import { loadConfig } from "./config.ts";
import { loadCredential } from "./enroll.ts";
import { pollOnce, uninstallSelf } from "./host.ts";
import { log } from "./log.ts";
import { Supervisor } from "./supervisor.ts";
import { threadWorkerFactory } from "./worker.ts";

async function main(): Promise<void> {
  const cfg = loadConfig();
  const credential = await loadCredential(cfg);
  log("info", "automations host starting", { serverUrl: cfg.serverUrl, name: cfg.hostName });

  const supervisor = new Supervisor(httpAutomationsApi, threadWorkerFactory, {
    serverUrl: cfg.serverUrl,
    timeoutMs: cfg.runTimeoutMs,
    heartbeatTimeoutMs: cfg.heartbeatTimeoutMs,
    memoryMb: cfg.memoryMb,
  });
  const deps = { api: httpAutomationsApi, supervisor, serverUrl: cfg.serverUrl, credential };

  let interval: ReturnType<typeof setInterval> | undefined;
  let removing = false;
  const tick = async (): Promise<void> => {
    if (removing) return;
    try {
      if ((await pollOnce(deps)) === "running") return;
    } catch (err) {
      log("error", "poll tick failed", { error: err instanceof Error ? err.message : String(err) });
      return;
    }
    removing = true;
    clearInterval(interval);
    await supervisor.stopAll();
    uninstallSelf(cfg.ctl, cfg.hostName);
    process.exit(0);
  };

  await tick();
  interval = setInterval(() => void tick(), cfg.pollMs);

  const shutdown = async (signal: string): Promise<void> => {
    log("info", "automations host shutting down", { signal });
    clearInterval(interval);
    await supervisor.stopAll();
    process.exit(0);
  };
  process.on("SIGTERM", () => void shutdown("SIGTERM"));
  process.on("SIGINT", () => void shutdown("SIGINT"));
}

main().catch((err) => {
  log("error", "automations host failed to start", { error: err instanceof Error ? err.message : String(err) });
  process.exit(1);
});
