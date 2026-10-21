import { describe, expect, it } from "bun:test";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import type { Automation } from "../src/define-automation.ts";
import type { ConfigSchema } from "../src/config-schema.ts";
import { scaffold } from "../bin/cli.ts";

const cli = path.resolve(import.meta.dir, "..", "bin", "cli.ts");

describe("the init scaffold", () => {
  it("bundles as push bundles it and imports where no nexul.config.json exists, as on a host", async () => {
    // Inside the package, so the scaffold's own "@otal-labs/nexul-sdk/..." imports resolve to this checkout.
    const dir = mkdtempSync(path.join(import.meta.dir, ".scaffold-"));
    try {
      const entry = path.join(dir, "index.ts");
      writeFileSync(entry, scaffold("Ticket logger", ""));
      writeFileSync(path.join(dir, "plays.generated.d.ts"), "export {};\n");
      const result = await Bun.build({ entrypoints: [entry], target: "node", format: "esm" });
      expect(result.success).toBe(true);
      const bundle = path.join(dir, "bundle.mjs");
      writeFileSync(bundle, await result.outputs[0]!.text());

      const mod = (await import(bundle)) as { default: Automation<ConfigSchema> };

      expect(mod.default.subscriptions).toEqual(["ticket.created"]);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});

describe("init", () => {
  const withInstance = async (status: number, check: (dir: string, url: string) => Promise<void>) => {
    const server = Bun.serve({
      port: 0,
      fetch: (req) =>
        req.headers.get("Authorization") === "Bearer dep_x" && new URL(req.url).pathname === "/api/automations/a1"
          ? Response.json({ name: "Ticket logger", description: "Logs new tickets" }, { status })
          : new Response("{}", { status: 404 }),
    });
    const dir = mkdtempSync(path.join(tmpdir(), "nexul-init-"));
    try {
      await check(dir, server.url.href);
    } finally {
      server.stop(true);
      rmSync(dir, { recursive: true, force: true });
    }
  };
  const initWithId = (dir: string, url: string) =>
    Bun.spawn(["bun", cli, "init", "--url", url, "--token", "dep_x", "--id", "a1"], { cwd: dir, stdin: "ignore", stdout: "ignore", stderr: "ignore" }).exited;

  it("fails before writing anything when the token can't read the automation", async () => {
    await withInstance(401, async (dir, url) => {
      expect(await initWithId(dir, url)).toBe(1);
      expect(existsSync(path.join(dir, "nexul.config.json"))).toBe(false);
    });
  });

  it("names the scaffold after the automation, since the code's name replaces it on connect", async () => {
    await withInstance(200, async (dir, url) => {
      expect(await initWithId(dir, url)).toBe(0);
      const entry = readFileSync(path.join(dir, "src", "index.ts"), "utf8");
      expect(entry).toContain('name: "Ticket logger"');
      expect(entry).toContain('description: "Logs new tickets"');
    });
  });

  it("takes the URL and token from flags, writes a tsconfig, and keeps the token file out of git once", () => {
    const dir = mkdtempSync(path.join(tmpdir(), "nexul-init-"));
    try {
      writeFileSync(path.join(dir, ".gitignore"), "node_modules");
      const init = () => Bun.spawnSync(["bun", cli, "init", "--url", "https://nexul.example.com/", "--token", "dep_x"], { cwd: dir, stdin: "ignore" });

      expect(init().exitCode).toBe(0);
      expect(init().exitCode).toBe(0);

      expect(JSON.parse(readFileSync(path.join(dir, "nexul.config.json"), "utf8"))).toEqual({ url: "https://nexul.example.com", token: "dep_x" });
      expect(readFileSync(path.join(dir, ".gitignore"), "utf8")).toBe("node_modules\nnexul.config.json\n");
      expect(existsSync(path.join(dir, "tsconfig.json"))).toBe(true);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});
