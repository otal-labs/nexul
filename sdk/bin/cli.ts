#!/usr/bin/env bun
// The package's own commands ("bunx @otal-labs/nexul-sdk init|dev|push|pull|types",
// no separate global tool). Deliberately one file: five small commands, each
// a handful of HTTP calls or fs writes — splitting them out would just be
// more files to jump between for no reuse gained.
import { appendFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import readline from "node:readline/promises";
import { stdin, stdout } from "node:process";
import { ApiClient } from "../src/api-client.ts";
import { eventFixtures, type Topic } from "../src/events.generated.ts";
import { createMockContext } from "../src/testing.ts";
import type { Automation } from "../src/define-automation.ts";
import type { ConfigSchema } from "../src/config-schema.ts";

const CONFIG_FILE = "nexul.config.json";
const ENTRY_FILE = path.join("src", "index.ts");
const PLAYS_FILE = path.join("src", "plays.generated.d.ts");

// The SDK ships TypeScript sources with .ts imports, which an editor resolves only under bundler resolution.
const TSCONFIG = `${JSON.stringify(
  {
    compilerOptions: {
      target: "ESNext",
      lib: ["ESNext", "DOM"],
      module: "Preserve",
      moduleResolution: "bundler",
      allowImportingTsExtensions: true,
      noEmit: true,
      strict: true,
      skipLibCheck: true,
    },
    include: ["src"],
  },
  null,
  2,
)}\n`;

interface ProjectConfig {
  url: string;
  token: string;
}

function loadConfig(): ProjectConfig {
  if (!existsSync(CONFIG_FILE)) {
    console.error(`no ${CONFIG_FILE} found here — run 'nexul init' first`);
    process.exit(1);
  }
  return JSON.parse(readFileSync(CONFIG_FILE, "utf8")) as ProjectConfig;
}

// node:readline auto-closes its Interface once non-TTY input hits EOF —
// which piped/redirected stdin does the instant it's fully written, often
// before every prompt() call has run. That makes a second question() throw
// ERR_USE_AFTER_CLOSE on anything but an interactive terminal (scripts,
// tests, CI). So: a TTY gets a real (reusable) readline interface; piped
// input is read to completion once and served line-by-line from memory.
let sharedRl: readline.Interface | undefined;
let pipedLines: string[] | undefined;

async function prompt(question: string): Promise<string> {
  if (!stdin.isTTY) {
    pipedLines ??= (await Bun.stdin.text()).split("\n");
    const line = (pipedLines.shift() ?? "").replace(/\r$/, "").trim();
    stdout.write(`${question}${line}\n`);
    return line;
  }
  sharedRl ??= readline.createInterface({ input: stdin, output: stdout });
  return (await sharedRl.question(question)).trim();
}

function closePrompts(): void {
  sharedRl?.close();
  sharedRl = undefined;
}

// The host imports the pushed bundle in a worker, so nothing here may run on import beyond defining the handlers.
export function scaffold(name: string, description: string): string {
  return `/// <reference path="./plays.generated.d.ts" />
import { defineAutomation } from "@otal-labs/nexul-sdk/automation";

const automation = defineAutomation({
  name: ${JSON.stringify(name)},
  description: ${JSON.stringify(description)},
  config: {},
});

automation.on("ticket.created", async (payload, ctx) => {
  ctx.log("ticket created", { id: payload.ticket.id });
  return true;
});

export default automation;
`;
}

// flag reads "--name value" from args, so the New automation dialog can pre-fill what init would ask.
function flag(args: string[], name: string): string | undefined {
  const at = args.indexOf(`--${name}`);
  if (at === -1) return undefined;
  return args[at + 1];
}

async function cmdInit(args: string[]): Promise<void> {
  const url = (flag(args, "url") ?? (await prompt("Nexul instance URL (e.g. https://deploy.example.com): "))).replace(/\/+$/, "");
  let token = flag(args, "token");
  if (!token) {
    console.log(`\nMint a personal access token at ${url}/settings/security?tab=tokens (Your settings -> Security -> Tokens), then paste it below.\n`);
    token = await prompt("Personal access token: ");
  }
  closePrompts();
  // The code's name and description replace the automation's when it connects, so start from the automation's own.
  const id = flag(args, "id");
  const named = id
    ? await new ApiClient({ baseUrl: url, token }).automations.get<{ name: string; description: string }>(id)
    : { name: "my-automation", description: "Describe what this automation does." };
  writeFileSync(CONFIG_FILE, `${JSON.stringify({ url, token }, null, 2)}\n`);
  ignoreConfig();
  if (!existsSync("tsconfig.json")) writeFileSync("tsconfig.json", TSCONFIG);
  if (!existsSync("src")) mkdirSync("src");
  if (!existsSync(ENTRY_FILE)) writeFileSync(ENTRY_FILE, scaffold(named.name, named.description));
  if (!existsSync(PLAYS_FILE)) writeFileSync(PLAYS_FILE, playsDeclaration([]));
  console.log(`\nWrote ${CONFIG_FILE} and ${ENTRY_FILE}.`);
  const target = id ?? "<automation-id>";
  console.log("Next: try your handlers on sample events, with no live effects, then upload the code as a pending version:");
  console.log("  bunx nexul dev");
  console.log(`  bunx nexul push ${target} "message"`);
}

// ignoreConfig keeps the personal access token in nexul.config.json out of git.
function ignoreConfig(): void {
  const current = existsSync(".gitignore") ? readFileSync(".gitignore", "utf8") : "";
  if (current.split("\n").some((line) => line.trim() === CONFIG_FILE)) return;
  const lead = current === "" || current.endsWith("\n") ? "" : "\n";
  appendFileSync(".gitignore", `${lead}${CONFIG_FILE}\n`);
}

async function loadAutomation(): Promise<Automation<ConfigSchema>> {
  if (!existsSync(ENTRY_FILE)) {
    console.error(`no ${ENTRY_FILE} found — run 'nexul init' first`);
    process.exit(1);
  }
  const mod = (await import(path.resolve(ENTRY_FILE))) as { default?: Automation<ConfigSchema> };
  if (!mod.default) {
    console.error(`${ENTRY_FILE} must 'export default' the result of defineAutomation(...)`);
    process.exit(1);
  }
  return mod.default;
}

async function cmdDev(): Promise<void> {
  const automation = await loadAutomation();
  const subs = automation.subscriptions;
  if (subs.length === 0) {
    console.log("no topics registered — call automation.on(topic, handler) in your automation first");
    return;
  }
  console.log(`${automation.name} — registered topics:`);
  subs.forEach((topic, i) => console.log(`  ${i + 1}. ${topic}`));
  const choice = await prompt("\nFire which topic? (number or exact name): ");
  closePrompts();
  const topic = (subs[Number(choice) - 1] ?? choice) as Topic;
  const handler = automation.getHandler(topic);
  if (!handler) {
    console.error(`no handler registered for topic ${topic}`);
    return;
  }
  const payload = eventFixtures[topic];
  console.log(`\nFiring fixture "${topic}":\n${JSON.stringify(payload, null, 2)}\n`);

  const ctx = createMockContext(automation.configSchema);
  let outcome: string;
  try {
    outcome = (await handler(payload, ctx)) === true ? "success" : "failure";
  } catch (err) {
    outcome = `crash: ${err instanceof Error ? err.message : String(err)}`;
  }

  console.log(`Outcome: ${outcome}`);
  console.log(ctx.logs.length ? "Logs:" : "Logs: (none)");
  for (const line of ctx.logs) console.log(`  ${line}`);
  console.log(ctx.calls.length ? "Would-have-called API (no live effects):" : "Would-have-called API: (none)");
  for (const call of ctx.calls) console.log(`  ${call.method} ${call.path}${call.body ? ` ${JSON.stringify(call.body)}` : ""}`);
}

async function cmdPush(automationId: string | undefined, message: string | undefined): Promise<void> {
  if (!automationId) {
    console.error("usage: nexul push <automation-id> [message]");
    process.exit(1);
  }
  const config = loadConfig();
  const entry = path.resolve(ENTRY_FILE);
  const result = await Bun.build({ entrypoints: [entry], target: "node", format: "esm" });
  if (!result.success) {
    for (const log of result.logs) console.error(String(log));
    process.exit(1);
  }
  const [output] = result.outputs;
  if (!output) {
    console.error("bundling produced no output");
    process.exit(1);
  }
  const code = await output.text();
  const source = readFileSync(ENTRY_FILE, "utf8");
  const api = new ApiClient({ baseUrl: config.url, token: config.token });
  await writePlayNames(api, automationId).catch((err: unknown) => {
    console.error(`could not refresh ${PLAYS_FILE}: ${err instanceof Error ? err.message : String(err)}`);
  });
  const version = await api.automations.versions.push<{ sequence: number }>(automationId, code, message, source);
  console.log(`Pushed version ${version.sequence} as pending. Merge it on the automation's Versions tab in Nexul.`);
}

async function cmdPull(automationId: string | undefined, versionId: string | undefined): Promise<void> {
  if (!automationId) {
    console.error("usage: nexul pull <automation-id> [version-id]");
    process.exit(1);
  }
  const config = loadConfig();
  const api = new ApiClient({ baseUrl: config.url, token: config.token });
  const version = versionId
    ? await api.automations.versions.get<PulledVersion>(automationId, versionId)
    : (await api.automations.versions.diff<{ active: PulledVersion | null }>(automationId)).active;
  if (!version) {
    console.error("automation has no active version to pull");
    process.exit(1);
  }
  // The bundle inlines the SDK, so writing it over src/index.ts would bury the author's code in it.
  if (!version.source) {
    console.error(`version ${version.sequence} was pushed before Nexul kept the source file, so there is nothing to pull; its bundle shows on the Versions tab`);
    process.exit(1);
  }
  if (!existsSync("src")) mkdirSync("src");
  writeFileSync(ENTRY_FILE, version.source);
  console.log(`wrote ${ENTRY_FILE}`);
}

interface PulledVersion {
  sequence: number;
  source?: string;
}

interface PlaySummary {
  label: string;
  type: string;
}

// playsDeclaration augments the SDK's PlayNames, so ctx.runPlay accepts exactly the workspace's ticket plays.
export function playsDeclaration(plays: PlaySummary[]): string {
  const lines = plays.map((p) => `    ${JSON.stringify(p.label)}: ${JSON.stringify(p.type)};`);
  return [
    "// Generated by `nexul types` from the workspace's plays; `nexul push` refreshes it. Do not edit.",
    "export {};",
    'declare module "@otal-labs/nexul-sdk/context" {',
    "  interface PlayNames {",
    ...lines,
    "  }",
    "}",
    "",
  ].join("\n");
}

async function writePlayNames(api: ApiClient, automationId: string): Promise<number> {
  const automation = await api.automations.get<{ workspace_id: string }>(automationId);
  const plays = await api.request<PlaySummary[]>("GET", `/api/workspaces/${encodeURIComponent(automation.workspace_id)}/plays`);
  if (!existsSync("src")) mkdirSync("src");
  writeFileSync(PLAYS_FILE, playsDeclaration(plays));
  return plays.length;
}

async function cmdTypes(automationId: string | undefined): Promise<void> {
  if (!automationId) {
    console.error("usage: nexul types <automation-id>");
    process.exit(1);
  }
  const config = loadConfig();
  const count = await writePlayNames(new ApiClient({ baseUrl: config.url, token: config.token }), automationId);
  console.log(`wrote ${PLAYS_FILE} with ${count} play names`);
}

async function main(): Promise<void> {
  const [command, ...rest] = process.argv.slice(2);
  switch (command) {
    case "init":
      await cmdInit(rest);
      return;
    case "dev":
      await cmdDev();
      return;
    case "push":
      await cmdPush(rest[0], rest[1]);
      return;
    case "pull":
      await cmdPull(rest[0], rest[1]);
      return;
    case "types":
      await cmdTypes(rest[0]);
      return;
    default:
      console.error("usage: nexul <init|dev|push|pull|types> [args]");
      process.exit(1);
  }
}

if (import.meta.main) {
  main().catch((err) => {
    console.error(err instanceof Error ? err.message : String(err));
    process.exit(1);
  });
}
