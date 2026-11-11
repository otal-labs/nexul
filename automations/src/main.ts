import { dirname, join } from "node:path";
import { httpAutomationsApi } from "./automations-api.ts";
import { loadConfig } from "./config.ts";
import { loadCredential } from "./enroll.ts";
import { pollOnce, uninstallSelf } from "./host.ts";
import { log, LogQueue, shipLogsTo } from "./log.ts";
import { httpPostLogs, LogShipper } from "./log-shipper.ts";
import { WorkspaceManagers } from "./managers.ts";
import { PoolResolution } from "./pool-resolution.ts";
import { OnePasswordResolver } from "./onepassword-resolver.ts";
import { SecretCheck, realClock } from "./secret-check.ts";
import { loadFingerprinter } from "./secret-fingerprints.ts";
import { Supervisor } from "./supervisor.ts";
import { takeWorkspaceTokens } from "./workspace-tokens.ts";
import { threadWorkerFactory } from "./worker.ts";

async function main(): Promise<void> {
  const logQueue = new LogQueue();
  shipLogsTo(logQueue);
  const cfg = loadConfig();
  const managers = new WorkspaceManagers(takeWorkspaceTokens(), (token) => new OnePasswordResolver(token));
  const credential = await loadCredential(cfg);
  const shipper = new LogShipper(logQueue, httpPostLogs(cfg.serverUrl, credential));
  shipper.start();
  log("info", "automations host starting", { serverUrl: cfg.serverUrl, name: cfg.hostName });

  const { fingerprinter, persisted } = loadFingerprinter(join(dirname(cfg.credentialFile), "secret-check.key"));
  log("info", persisted ? "secret check key is kept in the host's directory" : "secret check key is in memory only: the host's directory could not hold it");

  const supervisor = new Supervisor(
    httpAutomationsApi,
    threadWorkerFactory,
    {
      serverUrl: cfg.serverUrl,
      timeoutMs: cfg.runTimeoutMs,
      heartbeatTimeoutMs: cfg.heartbeatTimeoutMs,
      memoryMb: cfg.memoryMb,
    },
    new PoolResolution(managers),
    { fingerprinter, clock: realClock },
  );
  managers.start();
  const secretCheck = new SecretCheck(supervisor, managers, cfg.secretCheckIntervalMs);
  secretCheck.start();
  if (cfg.secretCheckIntervalMs === 0) log("info", "secret check is off");
  const deps = { api: httpAutomationsApi, supervisor, managers, serverUrl: cfg.serverUrl, credential };

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
    shipper.stop();
    clearInterval(interval);
    secretCheck.stop();
    await supervisor.stopAll();
    uninstallSelf(cfg.ctl, cfg.hostName);
    process.exit(0);
  };

  await tick();
  interval = setInterval(() => void tick(), cfg.pollMs);

  const shutdown = async (signal: string): Promise<void> => {
    log("info", "automations host shutting down", { signal });
    clearInterval(interval);
    secretCheck.stop();
    await supervisor.stopAll();
    shipper.stop();
    await shipper.flush();
    process.exit(0);
  };
  process.on("SIGTERM", () => void shutdown("SIGTERM"));
  process.on("SIGINT", () => void shutdown("SIGINT"));
}

main().catch((err) => {
  log("error", "automations host failed to start", { error: err instanceof Error ? err.message : String(err) });
  process.exit(1);
});
