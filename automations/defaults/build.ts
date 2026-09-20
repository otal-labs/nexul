#!/usr/bin/env bun
// Bundles each default automation the same way `nexul push` bundles a
// custom one (sdk/bin/cli.ts's cmdPush: Bun.build, target node, esm — one
// self-contained file, no external imports left to resolve at runtime) and
// writes the result where internal/automations/defaults.go go:embeds it.
// Run this after editing a default's index.ts; go:embed reads the committed
// output, it does not invoke bun at `go build` time.
import { mkdirSync } from "node:fs";
import path from "node:path";

const OUT_DIR = path.resolve(import.meta.dir, "..", "..", "internal", "automations", "defaults");

const DEFAULTS = [
  { name: "ticket-finished", entry: path.resolve(import.meta.dir, "ticket-finished", "index.ts") },
  { name: "pr-opened", entry: path.resolve(import.meta.dir, "pr-opened", "index.ts") },
];

async function main(): Promise<void> {
  mkdirSync(OUT_DIR, { recursive: true });
  for (const def of DEFAULTS) {
    const result = await Bun.build({ entrypoints: [def.entry], target: "node", format: "esm" });
    if (!result.success) {
      for (const log of result.logs) console.error(String(log));
      process.exit(1);
    }
    const [output] = result.outputs;
    if (!output) {
      console.error(`bundling ${def.name} produced no output`);
      process.exit(1);
    }
    const outFile = path.join(OUT_DIR, `${def.name}.bundle.js`);
    await Bun.write(outFile, await output.text());
    console.log(`wrote ${outFile}`);
  }
}

main();
