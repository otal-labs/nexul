import { describe, expect, spyOn, test } from "bun:test";
import { FakeResolver } from "./fake-resolver.ts";
import { WorkspaceManagers } from "../src/managers.ts";
import { PoolResolution } from "../src/pool-resolution.ts";

const acme = (name = "Ticket finished") => ({ workspaceId: "acme", automationId: "a1", automationName: name });
const globex = { workspaceId: "globex", automationId: "g1", automationName: "Deploy notice" };

async function managersOf(workspaces: Record<string, FakeResolver>): Promise<WorkspaceManagers> {
  const managers = new WorkspaceManagers(new Map(Object.keys(workspaces).map((w) => [w, `ops_${w}`])), (token) => workspaces[token.slice(4)] as FakeResolver);
  managers.start();
  await new Promise<void>((resolve) => setImmediate(resolve));
  return managers;
}

describe("PoolResolution", () => {
  test("a thread asking for another workspace's reference gets nothing, and the other workspace's token is never used", async () => {
    const acmeOp = new FakeResolver("op", { "op://acme/db/password": "acme-secret" });
    const globexOp = new FakeResolver("op", { "op://globex/db/password": "globex-secret" });
    const pool = new PoolResolution(await managersOf({ acme: acmeOp, globex: globexOp }), 0);

    const got = await pool.resolve(acme(), { OWN: "op://acme/db/password", THEIRS: "op://globex/db/password" });

    expect(got).toEqual({ OWN: "acme-secret" });
    expect(globexOp.calls).toEqual([]);
  });

  test("five automations of one workspace connecting together make one resolve call", async () => {
    const op = new FakeResolver("op", { "op://acme/db/password": "acme-secret", "op://acme/api/key": "key" });
    const pool = new PoolResolution(await managersOf({ acme: op }), 5);

    const got = await Promise.all(
      [1, 2, 3, 4, 5].map((n) => pool.resolve({ ...acme(`Automation ${n}`), automationId: `a${n}` }, { DB: "op://acme/db/password", API: "op://acme/api/key" })),
    );

    expect(op.calls).toHaveLength(1);
    expect(got.every((values) => values.DB === "acme-secret" && values.API === "key")).toBe(true);
  });

  test("a failed entry is absent and named in the log without its reference or any value", async () => {
    const op = new FakeResolver("op", {
      "op://acme/db/password": "acme-secret",
      "op://acme/db/missing": { error: "field_missing" },
      "op://acme/db/down": { error: "unreachable" },
    });
    const pool = new PoolResolution(await managersOf({ acme: op }), 0);
    const lines: string[] = [];
    const logSpy = spyOn(console, "log").mockImplementation((line: unknown) => lines.push(String(line)));

    try {
      const got = await pool.resolve(acme(), {
        OK: "op://acme/db/password",
        MISSING: "op://acme/db/missing",
        DOWN: "op://acme/db/down",
        PLAIN: "a literal the thread kept",
      });
      expect(got).toEqual({ OK: "acme-secret" });
    } finally {
      logSpy.mockRestore();
    }

    const log = lines.join("\n");
    expect(log).toContain("MISSING: field not found");
    expect(log).toContain("DOWN: 1Password unreachable");
    expect(log).not.toContain("op://");
    expect(log).not.toContain("acme-secret");
  });

  test("a workspace with no token on the host fails each of its references as having none", async () => {
    const pool = new PoolResolution(await managersOf({ acme: new FakeResolver("op", {}) }), 0);
    const lines: string[] = [];
    const logSpy = spyOn(console, "log").mockImplementation((line: unknown) => lines.push(String(line)));

    try {
      expect(await pool.resolve(globex, { DB: "op://globex/db/password" })).toEqual({});
    } finally {
      logSpy.mockRestore();
    }

    expect(lines.join("\n")).toContain("DB: no 1Password token for this workspace on the host");
  });

  test("a request that is not a map of references resolves nothing", async () => {
    const op = new FakeResolver("op", { "op://acme/db/password": "acme-secret" });
    const pool = new PoolResolution(await managersOf({ acme: op }), 0);

    expect(await pool.resolve(acme(), ["op://acme/db/password"])).toEqual({});
    expect(await pool.resolve(acme(), { A: 7, B: "not a reference" })).toEqual({});
    expect(op.calls).toEqual([]);
  });
});
