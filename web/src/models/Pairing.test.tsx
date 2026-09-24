import { describe, expect, it } from "vitest";

import { providerSetupLines, setupRunRows, setupRunning, type ComputerSetup, type SetupTurnSummary } from "@/models/Pairing";

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
