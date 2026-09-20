#!/usr/bin/env bun
// The package's own commands ("npx @nexul/sdk init|dev|push|pull",
// no separate global tool). Deliberately one file: four small commands, each
// a handful of HTTP calls or fs writes — splitting them out would just be
// more files to jump between for no reuse gained.
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
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

const SCAFFOLD = `import { defineAutomation } from "@nexul/sdk/automation";
import { DialinClient } from "@nexul/sdk/client";

const automation = defineAutomation({
  name: "my-automation",
  description: "Describe what this automation does.",
  config: {},
});

automation.on("ticket.created", async (payload, ctx) => {
  ctx.log("ticket created", { id: payload.ticket.id });
  return true;
});

export default automation;

if (import.meta.main) {
  const config = JSON.parse(await Bun.file("nexul.config.json").text()) as { url: string; token: string };
  await new DialinClient(automation, config).run();
}
`;

async function cmdInit(): Promise<void> {
  const url = (await prompt("Nexul instance URL (e.g. https://deploy.example.com): ")).replace(/\/+$/, "");
  console.log(`\nMint a personal access token at ${url}/settings?section=tokens (Settings -> Personal Access Tokens), then paste it below.\n`);
  const token = await prompt("Personal access token: ");
  closePrompts();
  writeFileSync(CONFIG_FILE, `${JSON.stringify({ url, token }, null, 2)}\n`);
  if (!existsSync("src")) mkdirSync("src");
  if (!existsSync(ENTRY_FILE)) writeFileSync(ENTRY_FILE, SCAFFOLD);
  console.log(`\nWrote ${CONFIG_FILE} and ${ENTRY_FILE}.`);
  console.log("Next: create the automation itself in the Nexul UI (New automation) to get its id, then:");
  console.log("  nexul dev            # fire fixture events at your handlers, no live effects");
  console.log("  nexul push <id>      # bundle and upload as a pending version");
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
  const api = new ApiClient({ baseUrl: config.url, token: config.token });
  const version = await api.automations.versions.push(automationId, code, message);
  console.log(`pushed pending version: ${JSON.stringify(version)}`);
}

async function cmdPull(automationId: string | undefined, versionId: string | undefined): Promise<void> {
  if (!automationId) {
    console.error("usage: nexul pull <automation-id> [version-id]");
    process.exit(1);
  }
  const config = loadConfig();
  const api = new ApiClient({ baseUrl: config.url, token: config.token });
  const version = versionId
    ? await api.automations.versions.get<{ code: string }>(automationId, versionId)
    : (await api.automations.versions.diff<{ active: { code: string } | null }>(automationId)).active;
  if (!version) {
    console.error("automation has no active version to pull");
    process.exit(1);
  }
  if (!existsSync("src")) mkdirSync("src");
  writeFileSync(ENTRY_FILE, version.code);
  console.log(`wrote ${ENTRY_FILE}`);
}

async function main(): Promise<void> {
  const [command, ...rest] = process.argv.slice(2);
  switch (command) {
    case "init":
      await cmdInit();
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
    default:
      console.error("usage: nexul <init|dev|push|pull> [args]");
      process.exit(1);
  }
}

if (import.meta.main) {
  main().catch((err) => {
    console.error(err instanceof Error ? err.message : String(err));
    process.exit(1);
  });
}
