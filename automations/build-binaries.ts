import type { BunPlugin } from "bun";

// Compiles the host into one self-contained binary per release target, named like the Go assets
// (nexul-automations-<os>-<arch>[.exe]); the worker entrypoint is bundled too, so workers run from inside it.
const targets = [
  ["bun-linux-x64", "linux-amd64"],
  ["bun-linux-arm64", "linux-arm64"],
  ["bun-darwin-x64", "darwin-amd64"],
  ["bun-darwin-arm64", "darwin-arm64"],
  ["bun-windows-x64", "windows-amd64.exe"],
] as const;

// The 1Password core reads its wasm from __dirname at import, which a compiled binary resolves to the build
// machine's node_modules; requiring the file instead embeds it in the binary.
const wasmLoad = "require('path').join(__dirname, 'core_bg.wasm')";
const embedOnePasswordWasm: BunPlugin = {
  name: "embed-1password-wasm",
  setup(build) {
    build.onLoad({ filter: /@1password[\\/]sdk-core[\\/]nodejs[\\/]core\.js$/ }, async (args) => {
      const source = await Bun.file(args.path).text();
      if (!source.includes(wasmLoad)) throw new Error(`${args.path} no longer loads its wasm as ${wasmLoad}`);
      return { loader: "js", contents: source.replace(wasmLoad, "require('./core_bg.wasm')") };
    });
  },
};

const only = process.argv[2];
// The release workflow sets NEXUL_VERSION to the tag, so the host reports the release it was built for.
const define = { NEXUL_HOST_VERSION: JSON.stringify(process.env.NEXUL_VERSION ?? "dev") };
for (const [target, suffix] of targets) {
  if (only && !suffix.startsWith(only)) continue;
  // A failed build throws, which exits non-zero with the bundler's messages.
  await Bun.build({
    entrypoints: [`${import.meta.dir}/src/main.ts`, `${import.meta.dir}/src/worker-entry.ts`],
    compile: { target, outfile: `${import.meta.dir}/dist/nexul-automations-${suffix}` },
    define,
    plugins: [embedOnePasswordWasm],
  });
}
