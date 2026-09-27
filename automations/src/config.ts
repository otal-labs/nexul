export interface HostConfig {
  serverUrl: string;
  credentialFile: string;
  // hostName is this host's unit name, used only to uninstall itself once removed.
  hostName: string;
  ctl: string;
  // enrollCodeFile is set only by dev stacks: a first start with no credential enrolls with the code in it.
  enrollCodeFile: string | null;
  pollMs: number;
  memoryMb: number;
  heartbeatTimeoutMs: number;
  runTimeoutMs: number;
}

function envInt(env: NodeJS.ProcessEnv, name: string, fallback: number): number {
  const raw = env[name];
  if (!raw) return fallback;
  const n = Number(raw);
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

function required(env: NodeJS.ProcessEnv, name: string): string {
  const value = env[name];
  if (!value) throw new Error(`${name} is required`);
  return value;
}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): HostConfig {
  return {
    serverUrl: required(env, "NEXUL_SERVER_URL").replace(/\/+$/, ""),
    credentialFile: required(env, "NEXUL_CREDENTIAL_FILE"),
    hostName: required(env, "NEXUL_AUTOMATIONS_HOST_NAME"),
    ctl: env.NEXUL_CTL || "nexul",
    enrollCodeFile: env.NEXUL_ENROLL_CODE_FILE || null,
    pollMs: envInt(env, "NEXUL_AUTOMATIONS_POLL_MS", 10_000),
    memoryMb: envInt(env, "NEXUL_AUTOMATIONS_MEMORY_MB", 128),
    // heartbeatTimeoutMs is the host-level watchdog (research doc §1/§4): a
    // blocked event loop never lets the SDK's own async run-timeout fire, so
    // this is the external backstop that terminates the worker outright.
    heartbeatTimeoutMs: envInt(env, "NEXUL_AUTOMATIONS_HEARTBEAT_TIMEOUT_MS", 90_000),
    runTimeoutMs: envInt(env, "NEXUL_AUTOMATIONS_RUN_TIMEOUT_MS", 30_000),
  };
}
