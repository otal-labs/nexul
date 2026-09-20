import { httpAutomationsApi } from "./automations-api.ts";
import { loadConfig } from "./config.ts";
import { readHostTokens } from "./host-tokens.ts";
import { log } from "./log.ts";
import { Supervisor } from "./supervisor.ts";
import { threadWorkerFactory } from "./worker.ts";

async function main(): Promise<void> {
  const cfg = loadConfig();
  const targets = readHostTokens(cfg.tokensPath);
  log("info", "automations host starting", { serverUrl: cfg.serverUrl, automations: targets.length });

  const supervisor = new Supervisor(targets, httpAutomationsApi, threadWorkerFactory, {
    serverUrl: cfg.serverUrl,
    timeoutMs: cfg.runTimeoutMs,
    heartbeatTimeoutMs: cfg.heartbeatTimeoutMs,
    memoryMb: cfg.memoryMb,
  });
  await supervisor.pollOnce();
  const interval = setInterval(() => {
    supervisor.pollOnce().catch((err) => log("error", "poll tick failed", { error: err instanceof Error ? err.message : String(err) }));
  }, cfg.pollMs);

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
