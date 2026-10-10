import { describe, expect, it } from "bun:test";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { playsDeclaration } from "../bin/cli.ts";

const sdk = path.resolve(import.meta.dir, "..");
const tsc = path.join(sdk, "node_modules", ".bin", "tsc");

// typecheck compiles files as an author's project would, against the SDK's own context module.
function typecheck(files: Record<string, string>): { exitCode: number; output: string } {
  const dir = mkdtempSync(path.join(tmpdir(), "nexul-sdk-types-"));
  try {
    const tsconfig = {
      compilerOptions: {
        target: "ES2022",
        lib: ["ES2022", "DOM"],
        module: "ESNext",
        moduleResolution: "bundler",
        strict: true,
        exactOptionalPropertyTypes: true,
        allowImportingTsExtensions: true,
        noEmit: true,
        skipLibCheck: true,
        types: [],
        paths: { "@nexul/sdk/context": [path.join(sdk, "src", "context.ts")] },
      },
      include: ["*.ts"],
    };
    writeFileSync(path.join(dir, "tsconfig.json"), JSON.stringify(tsconfig));
    for (const [name, text] of Object.entries(files)) writeFileSync(path.join(dir, name), text);
    const run = Bun.spawnSync([tsc, "-p", dir]);
    return { exitCode: run.exitCode, output: run.stdout.toString() + run.stderr.toString() };
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}

const handler = `import type { Ctx } from "@nexul/sdk/context";
export const run = async (ctx: Ctx) => {
  await ctx.runPlay("Fix with AI", "ticket-1", { runOn: "tester", priority: "high" });
  await ctx.runPlay('Say "hi"', "ticket-1");
  // @ts-expect-error a typo names no play
  await ctx.runPlay("Fix with Al", "ticket-1");
  // @ts-expect-error a doc play does not run on a ticket
  await ctx.runPlay("To tickets via AI", "ticket-1");
};
`;

describe("runPlay's play names", () => {
  it("accept any label before nexul types has run", () => {
    const fresh = `import type { Ctx } from "@nexul/sdk/context";
export const run = (ctx: Ctx) => ctx.runPlay("Anything at all", "ticket-1");
`;
    const result = typecheck({ "index.ts": fresh, "plays.generated.d.ts": playsDeclaration([]) });
    expect(result.output).toBe("");
    expect(result.exitCode).toBe(0);
  });

  it("accept only the workspace's ticket plays once generated", () => {
    const names = playsDeclaration([
      { label: "Fix with AI", type: "ticket" },
      { label: "To tickets via AI", type: "doc" },
      { label: 'Say "hi"', type: "ticket" },
      { label: "Interview", type: "interview" },
    ]);
    const result = typecheck({ "index.ts": handler, "plays.generated.d.ts": names });
    expect(result.output).toBe("");
    expect(result.exitCode).toBe(0);
  });

  it("are what makes a typo fail: without them the same handler's expected errors go unused", () => {
    const result = typecheck({ "index.ts": handler, "plays.generated.d.ts": playsDeclaration([]) });
    expect(result.output).toContain("Unused '@ts-expect-error' directive");
    expect(result.exitCode).not.toBe(0);
  });
});
