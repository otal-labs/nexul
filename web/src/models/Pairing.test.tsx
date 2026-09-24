import { describe, expect, it } from "vitest";

import {
  providerSetupLines,
  setupModelChoices,
  setupRunRows,
  setupRunning,
  type ComputerSetup,
  type HarnessProvider,
  type SetupTurnSummary,
} from "@/models/Pairing";

const turn = (overrides: Partial<SetupTurnSummary>): SetupTurnSummary => ({
  run_id: "r0",
  turn_id: "t0",
  provider: "codex",
  provider_name: "Codex",
  state: "confirmed",
  status: "Confirmed with 12 skills",
  updated_at: "2026-09-24T00:00:00Z",
  ...overrides,
});

describe("setupRunRows", () => {
  it("shows each provider's newest turn when no run was started here", () => {
    const rows = setupRunRows([turn({}), turn({ provider: "opencode", provider_name: "", state: "failed", turn_id: "t1" })], undefined);
    expect(rows.map((r) => [r.name, r.state])).toEqual([
      ["Codex", "confirmed"],
      ["opencode", "failed"],
    ]);
    expect(setupRunning(rows)).toBe(false);
  });

  it("lists the run's providers in order, queued until their turn in this run starts", () => {
    const run = {
      run_id: "r1",
      computer_id: "c1",
      providers: [
        { provider: "claudeagent", name: "Claude" },
        { provider: "codex", name: "Codex" },
      ],
    };
    const rows = setupRunRows([turn({}), turn({ run_id: "r1", turn_id: "t2", provider: "claudeagent", provider_name: "Claude", state: "running" })], run);
    expect(rows.map((r) => [r.provider, r.state, r.turnId])).toEqual([
      ["claudeagent", "running", "t2"],
      ["codex", "queued", undefined],
    ]);
    expect(setupRunning(rows)).toBe(true);
  });

  it("carries the model each turn ran on, or the one the run just picked, empty for the provider default", () => {
    const run = { run_id: "r1", computer_id: "c1", providers: [{ provider: "claudeagent", name: "Claude", model: "claude-haiku" }] };
    const rows = setupRunRows([turn({ model: "gpt-mini" }), turn({ provider: "opencode", turn_id: "t1" })], run);
    expect(rows.map((r) => [r.provider, r.model])).toEqual([
      ["claudeagent", "claude-haiku"],
      ["codex", "gpt-mini"],
      ["opencode", ""],
    ]);
  });

  it("keeps the other providers' rows when a retry covers only one", () => {
    const run = { run_id: "r2", computer_id: "c1", providers: [{ provider: "opencode", name: "opencode" }] };
    const rows = setupRunRows([turn({}), turn({ provider: "opencode", state: "failed", turn_id: "t1" })], run);
    expect(rows.map((r) => [r.provider, r.state])).toEqual([
      ["opencode", "queued"],
      ["codex", "confirmed"],
    ]);
  });
});

describe("providerSetupLines", () => {
  const setup = (overrides: Partial<ComputerSetup>): ComputerSetup => ({ computer_id: "c1", confirmed_at: null, providers: [], turns: [], ...overrides });

  it("joins confirmations and turns into one line per provider, a running turn first", () => {
    const lines = providerSetupLines(
      setup({
        providers: [
          { provider: "codex", confirmed_at: "2026-09-20T00:00:00Z", skills: ["tdd"] },
          { provider: "opencode", confirmed_at: null, skills: [] },
        ],
        turns: [
          turn({ provider: "codex", state: "running" }),
          turn({ provider: "claudeagent", provider_name: "Claude", state: "failed" }),
        ],
      }),
    );
    expect(lines.map((l) => [l.name, l.state])).toEqual([
      ["Codex", "running"],
      ["opencode", "unconfirmed"],
      ["Claude", "failed"],
    ]);
  });

  it("keeps a provider confirmed after a later failed re-run, with its confirmed-at time", () => {
    const lines = providerSetupLines(
      setup({ providers: [{ provider: "codex", confirmed_at: "2026-09-20T00:00:00Z", skills: [] }], turns: [turn({ state: "failed" })] }),
    );
    expect(lines).toEqual([{ provider: "codex", name: "Codex", state: "confirmed", confirmedAt: "2026-09-20T00:00:00Z" }]);
  });
});

describe("setupModelChoices", () => {
  const provider = (id: string, driver: string, models: HarnessProvider["models"]): HarnessProvider => ({ id, driver, name: id, models, needs_setup: true });
  const providers = [
    provider("claude", "claudeAgent", [
      { slug: "claude-big", name: "Big" },
      { slug: "claude-small", name: "Small", is_default: true },
    ]),
    provider("claude-work", "claudeAgent", []),
    provider("opencode", "opencode", [
      { slug: "pickle", name: "Pickle", is_default: true },
      { slug: "gpt", name: "GPT" },
    ]),
    provider("grok", "grok", [{ slug: "grok-1", name: "Grok 1" }]),
  ];

  it("offers one choice per driver, the defaults' model on the default provider and each other provider's own default", () => {
    const choices = setupModelChoices(providers, { provider: "claude", model: "claude-big" });
    expect(choices.map((c) => [c.provider, c.preselected])).toEqual([
      ["claudeagent", "claude-big"],
      ["opencode", "pickle"],
      ["grok", ""],
    ]);
  });

  it("falls back to the provider's own default when the defaults name a model it does not list", () => {
    expect(setupModelChoices(providers, { provider: "claude", model: "gone" })[0]?.preselected).toBe("claude-small");
    expect(setupModelChoices(providers, undefined)[0]?.preselected).toBe("claude-small");
  });
});
