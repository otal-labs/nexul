import { afterAll, expect, test } from "bun:test";
import { cpSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const pkg = join(import.meta.dir, "..");
const work = mkdtempSync(join(tmpdir(), "nexul-host-build-"));
afterAll(() => rmSync(work, { recursive: true, force: true }));

const os = process.platform === "win32" ? "windows" : process.platform;
const arch = process.arch === "x64" ? "amd64" : process.arch;
const suffix = `${os}-${arch}${os === "windows" ? ".exe" : ""}`;

// A release binary runs on a machine without the build's node_modules, so the build happens in a copy that is
// deleted before the binary starts.
test("the compiled host starts on a machine without the build's node_modules", () => {
  for (const entry of ["build-binaries.ts", "package.json", "src", "node_modules"]) {
    cpSync(join(pkg, entry), join(work, entry), { recursive: true, dereference: true });
  }
  const build = Bun.spawnSync([process.execPath, "run", "build-binaries.ts", suffix], { cwd: work, stderr: "pipe" });
  expect(build.exitCode, build.stderr.toString()).toBe(0);
  rmSync(join(work, "node_modules"), { recursive: true, force: true });

  const run = Bun.spawnSync([join(work, "dist", `nexul-automations-${suffix}`)], { cwd: work, env: {}, stderr: "pipe" });

  expect(run.stderr.toString()).toContain("NEXUL_SERVER_URL is required");
}, 120_000);
