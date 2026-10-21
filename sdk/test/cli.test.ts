import { describe, expect, it } from "bun:test";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
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

// withInstance runs check against a fake instance answering with route, in a fresh project folder under parent.
async function withInstance(route: (req: Request) => Response | Promise<Response>, check: (dir: string, url: string) => Promise<void>, parent = tmpdir()) {
  const server = Bun.serve({ port: 0, fetch: route });
  const dir = mkdtempSync(path.join(parent, ".nexul-cli-"));
  try {
    await check(dir, server.url.href);
  } finally {
    server.stop(true);
    rmSync(dir, { recursive: true, force: true });
  }
}

const run = (dir: string, ...args: string[]) => Bun.spawn(["bun", cli, ...args], { cwd: dir, stdin: "ignore", stdout: "ignore", stderr: "ignore" }).exited;

const project = (dir: string, url: string, entry: string) => {
  writeFileSync(path.join(dir, "nexul.config.json"), JSON.stringify({ url, token: "dep_x" }));
  mkdirSync(path.join(dir, "src"));
  writeFileSync(path.join(dir, "src", "index.ts"), entry);
};

describe("init", () => {
  const automation = (status: number) => (req: Request) =>
    req.headers.get("Authorization") === "Bearer dep_x" && new URL(req.url).pathname === "/api/automations/a1"
      ? Response.json({ name: "Ticket logger", description: "Logs new tickets" }, { status })
      : new Response("{}", { status: 404 });
  const initWithId = (dir: string, url: string) => run(dir, "init", "--url", url, "--token", "dep_x", "--id", "a1");

  it("fails before writing anything when the token can't read the automation", async () => {
    await withInstance(automation(401), async (dir, url) => {
      expect(await initWithId(dir, url)).toBe(1);
      expect(existsSync(path.join(dir, "nexul.config.json"))).toBe(false);
    });
  });

  it("names the scaffold after the automation, since the code's name replaces it on connect", async () => {
    await withInstance(automation(200), async (dir, url) => {
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

describe("push and pull", () => {
  it("push sends the source file beside the bundle the host runs", async () => {
    let body: { code: string; source: string } | undefined;
    const route = async (req: Request) => {
      if (req.method !== "POST") return new Response("{}", { status: 404 });
      body = (await req.json()) as { code: string; source: string };
      return Response.json({ sequence: 1 }, { status: 201 });
    };
    const entry = scaffold("Ticket logger", "");
    // Inside the package, so the bundler resolves the scaffold's SDK imports to this checkout.
    await withInstance(route, async (dir, url) => {
      project(dir, url, entry);
      writeFileSync(path.join(dir, "src", "plays.generated.d.ts"), "export {};\n");

      expect(await run(dir, "push", "a1", "first")).toBe(0);

      expect(body?.source).toBe(entry);
      expect(body?.code).toContain("class Automation");
    }, import.meta.dir);
  });

  const active = (version: object) => () => Response.json({ active: version });

  it("pull writes the pushed source back, not the bundle", async () => {
    await withInstance(active({ sequence: 2, code: "bundle", source: "the source" }), async (dir, url) => {
      project(dir, url, "old");

      expect(await run(dir, "pull", "a1")).toBe(0);

      expect(readFileSync(path.join(dir, "src", "index.ts"), "utf8")).toBe("the source");
    });
  });

  it("pull leaves src/index.ts alone for a version pushed before sources were kept", async () => {
    await withInstance(active({ sequence: 1, code: "bundle" }), async (dir, url) => {
      project(dir, url, "mine");

      expect(await run(dir, "pull", "a1")).toBe(1);

      expect(readFileSync(path.join(dir, "src", "index.ts"), "utf8")).toBe("mine");
    });
  });
});
