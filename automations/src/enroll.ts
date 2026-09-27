import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
import type { HostConfig } from "./config.ts";
import { log } from "./log.ts";

// EnrollRefused is a refusal retrying cannot fix: the server read the code and said no.
export class EnrollRefused extends Error {
  constructor(message: string) {
    super(message);
    this.name = "EnrollRefused";
  }
}

export interface EnrollDeps {
  fetch: (url: string, init: RequestInit) => Promise<Response>;
  sleep: (ms: number) => Promise<void>;
  now: () => number;
  waitMs: number;
  retryMs: number;
}

const defaultDeps: EnrollDeps = {
  fetch: (url, init) => fetch(url, init),
  sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
  now: () => Date.now(),
  waitMs: 5 * 60_000,
  retryMs: 1_000,
};

// Go's GOOS/GOARCH spelling, which the server and the release assets use.
export const hostOS = (platform: string = process.platform): string => (platform === "win32" ? "windows" : platform);
export const hostArch = (arch: string = process.arch): string => (arch === "x64" ? "amd64" : arch);

// loadCredential reads the host's credential. A dev stack's first start has none yet: it enrolls with the code the
// server writes at boot, retrying until the file appears and the server answers, and writes the credential (0600).
export async function loadCredential(cfg: HostConfig, deps: EnrollDeps = defaultDeps): Promise<string> {
  if (existsSync(cfg.credentialFile)) return readFileSync(cfg.credentialFile, "utf8").trim();
  if (!cfg.enrollCodeFile) {
    throw new Error(`${cfg.credentialFile} does not exist; install this host with nexul install automations`);
  }
  const deadline = deps.now() + deps.waitMs;
  for (;;) {
    let lastError: unknown;
    try {
      const credential = await enrollOnce(cfg, cfg.enrollCodeFile, deps);
      mkdirSync(dirname(cfg.credentialFile), { recursive: true, mode: 0o700 });
      writeFileSync(cfg.credentialFile, `${credential}\n`, { mode: 0o600 });
      log("info", "automations host enrolled", { name: cfg.hostName });
      return credential;
    } catch (err) {
      if (err instanceof EnrollRefused) throw err;
      lastError = err;
    }
    if (deps.now() >= deadline) {
      throw new Error(`enroll with the code in ${cfg.enrollCodeFile}: ${lastError instanceof Error ? lastError.message : String(lastError)}`);
    }
    await deps.sleep(deps.retryMs);
  }
}

async function enrollOnce(cfg: HostConfig, codeFile: string, deps: EnrollDeps): Promise<string> {
  const code = readFileSync(codeFile, "utf8").trim();
  const res = await deps.fetch(`${cfg.serverUrl}/api/automation-hosts/enroll`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code, name: cfg.hostName, os: hostOS(), arch: hostArch(), version: "dev" }),
  });
  const text = await res.text();
  if (res.status >= 500) throw new Error(`the server answered ${res.status}`);
  if (!res.ok) throw new EnrollRefused(`the server refused the enrollment: ${res.status} ${text.trim()}`);
  const body = JSON.parse(text) as { credential?: unknown };
  if (typeof body.credential !== "string" || body.credential === "") {
    throw new EnrollRefused("the server's answer holds no credential");
  }
  return body.credential;
}
