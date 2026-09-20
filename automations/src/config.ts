export interface HostConfig {
  serverUrl: string;
  tokensPath: string;
  pollMs: number;
  memoryMb: number;
  heartbeatTimeoutMs: number;
  runTimeoutMs: number;
}

function envInt(name: string, fallback: number): number {
  const raw = process.env[name];
  if (!raw) return fallback;
  const n = Number(raw);
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): HostConfig {
  const serverUrl = env.NEXUL_SERVER_URL;
  if (!serverUrl) throw new Error("NEXUL_SERVER_URL is required");
  return {
    serverUrl: serverUrl.replace(/\/+$/, ""),
    tokensPath: env.NEXUL_AUTOMATIONS_TOKENS_PATH ?? "/data/automations-tokens.json",
    pollMs: envInt("NEXUL_AUTOMATIONS_POLL_MS", 10_000),
    memoryMb: envInt("NEXUL_AUTOMATIONS_MEMORY_MB", 128),
    // heartbeatTimeoutMs is the host-level watchdog (research doc §1/§4): a
    // blocked event loop never lets the SDK's own async run-timeout fire, so
    // this is the external backstop that terminates the worker outright.
    heartbeatTimeoutMs: envInt("NEXUL_AUTOMATIONS_HEARTBEAT_TIMEOUT_MS", 90_000),
    runTimeoutMs: envInt("NEXUL_AUTOMATIONS_RUN_TIMEOUT_MS", 30_000),
  };
}
