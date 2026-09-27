import { describe, expect, test } from "bun:test";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { HostConfig } from "../src/config.ts";
import { EnrollRefused, hostArch, hostOS, loadCredential, type EnrollDeps } from "../src/enroll.ts";

function config(overrides: Partial<HostConfig> = {}): HostConfig {
  const dir = mkdtempSync(join(tmpdir(), "enroll-test-"));
  return {
    serverUrl: "http://server:8080",
    credentialFile: join(dir, "state", "credential"),
    hostName: "instance",
    ctl: "nexul",
    enrollCodeFile: join(dir, "automations-instance"),
    pollMs: 10_000,
    memoryMb: 128,
    heartbeatTimeoutMs: 90_000,
    runTimeoutMs: 30_000,
    ...overrides,
  };
}

// fakeDeps runs on a virtual clock: sleeping advances it, and onSleep lets a test act while "time passes".
function fakeDeps(answers: (() => Promise<Response>)[], onSleep: () => void = () => {}): EnrollDeps & { bodies: unknown[] } {
  let clock = 0;
  const bodies: unknown[] = [];
  return {
    bodies,
    fetch: async (url, init) => {
      expect(url).toBe("http://server:8080/api/automation-hosts/enroll");
      bodies.push(JSON.parse(String(init.body)));
      const next = answers.shift();
      if (!next) throw new Error("no more answers");
      return next();
    },
    sleep: async (ms) => {
      clock += ms;
      onSleep();
    },
    now: () => clock,
    waitMs: 10_000,
    retryMs: 1_000,
  };
}

describe("loadCredential", () => {
  test("refuses to start with no credential and nothing to enroll with", async () => {
    await expect(loadCredential(config({ enrollCodeFile: null }), fakeDeps([]))).rejects.toThrow("nexul install automations");
  });

  test("a refused code fails at once, writing nothing", async () => {
    const cfg = config();
    writeFileSync(cfg.enrollCodeFile as string, "nxe_used");
    const deps = fakeDeps([async () => new Response('{"code":"invalid_code"}', { status: 401 })]);

    await expect(loadCredential(cfg, deps)).rejects.toBeInstanceOf(EnrollRefused);
    expect(existsSync(cfg.credentialFile)).toBe(false);
  });

  test("an answer without a credential is refused", async () => {
    const cfg = config();
    writeFileSync(cfg.enrollCodeFile as string, "nxe_abc");

    await expect(loadCredential(cfg, fakeDeps([async () => new Response("{}", { status: 201 })]))).rejects.toBeInstanceOf(EnrollRefused);
  });

  test("gives up once the code never appears", async () => {
    await expect(loadCredential(config(), fakeDeps([]))).rejects.toThrow("enroll with the code in");
  });

  test("an empty code file is still being written, so it waits instead of sending an empty code", async () => {
    const cfg = config();
    writeFileSync(cfg.enrollCodeFile!, "");
    const deps = fakeDeps([async () => new Response(JSON.stringify({ credential: "nxa_new" }), { status: 201 })], () =>
      writeFileSync(cfg.enrollCodeFile!, "nxe_abc"),
    );
    expect(await loadCredential(cfg, deps)).toBe("nxa_new");
    expect(deps.bodies).toHaveLength(1);
    expect((deps.bodies[0] as { code: string }).code).toBe("nxe_abc");
  });

  test("reads an existing credential without enrolling", async () => {
    const cfg = config();
    writeFileSync(cfg.enrollCodeFile as string, "nxe_abc");
    mkdirSync(join(cfg.credentialFile, ".."), { recursive: true });
    writeFileSync(cfg.credentialFile, "nxa_existing\n");

    expect(await loadCredential(cfg, fakeDeps([]))).toBe("nxa_existing");
  });

  test("waits for the code and a server still booting, then writes the credential", async () => {
    const cfg = config();
    let sleeps = 0;
    const deps = fakeDeps(
      [
        async () => {
          throw new Error("connection refused");
        },
        async () => new Response("", { status: 503 }),
        async () => new Response('{"id":"h1","credential":"nxa_new"}', { status: 201 }),
      ],
      () => {
        sleeps += 1;
        if (sleeps === 2) writeFileSync(cfg.enrollCodeFile as string, "nxe_abc\n");
      },
    );

    expect(await loadCredential(cfg, deps)).toBe("nxa_new");
    expect(readFileSync(cfg.credentialFile, "utf8")).toBe("nxa_new\n");
    expect(statSync(cfg.credentialFile).mode & 0o777).toBe(0o600);
    expect(deps.bodies).toHaveLength(3);
    expect(deps.bodies[0]).toEqual({ code: "nxe_abc", name: "instance", os: hostOS(), arch: hostArch(), version: "dev" });
  });
});

describe("platform names", () => {
  test("are spelled the way the Go release assets are", () => {
    expect(hostOS("win32")).toBe("windows");
    expect(hostOS("darwin")).toBe("darwin");
    expect(hostArch("x64")).toBe("amd64");
    expect(hostArch("arm64")).toBe("arm64");
  });
});
