// Compiles the host into one self-contained binary per release target, named like the Go assets
// (nexul-automations-<os>-<arch>[.exe]); the worker entrypoint is bundled too, so workers run from inside it.
const targets = [
  ["bun-linux-x64", "linux-amd64"],
  ["bun-linux-arm64", "linux-arm64"],
  ["bun-darwin-x64", "darwin-amd64"],
  ["bun-darwin-arm64", "darwin-arm64"],
  ["bun-windows-x64", "windows-amd64.exe"],
] as const;

const only = process.argv[2];
for (const [target, suffix] of targets) {
  if (only && !suffix.startsWith(only)) continue;
  const outfile = `dist/nexul-automations-${suffix}`;
  const build = Bun.spawnSync(
    [process.execPath, "build", "--compile", `--target=${target}`, "./src/main.ts", "./src/worker-entry.ts", "--outfile", outfile],
    { cwd: import.meta.dir, stdout: "inherit", stderr: "inherit" },
  );
  if (build.exitCode !== 0) {
    process.exit(build.exitCode ?? 1);
  }
}
