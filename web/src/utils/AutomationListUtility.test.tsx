import { describe, expect, it } from "vitest";

import { AutomationKind, AutomationRunOutcome } from "@/enums/Automation";
import { AutomationAttention, AutomationState, type Automation } from "@/models/Automation";
import { automationGroups, automationLastRunText, automationListItems, automationWeekText } from "@/utils/AutomationListUtility";

const failedRun = { outcome: AutomationRunOutcome.Failure, started_at: new Date().toISOString(), error: "boom" };

const automation = (name: string, overrides: Partial<Automation> = {}): Automation => ({
  id: name,
  workspace_id: "ws-1",
  name,
  description: "",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: [],
  config_schema: {},
  config_values: {},
  scopes: [],
  host_id: null,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

describe("automationListItems", () => {
  it.each([
    { name: "off wins over a failed last run", a: automation("x", { enabled: false, last_run: failedRun }), state: AutomationState.Off },
    { name: "a failed last run is failing", a: automation("x", { last_run: failedRun, pending_version: 3 }), state: AutomationState.Failing },
    { name: "a pending version needs attention", a: automation("x", { pending_version: 3 }), state: AutomationState.Attention },
    { name: "an offline host needs attention", a: automation("x", { host_id: "h-off" }), state: AutomationState.Attention },
    { name: "an online host is on", a: automation("x", { host_id: "h-on" }), state: AutomationState.On },
  ])("$name", ({ a, state }) => {
    expect(automationListItems([a], new Set(["h-off"]))[0]?.state).toBe(state);
  });

  it("names an offline host before a pending version", () => {
    const [item] = automationListItems([automation("x", { host_id: "h-off", pending_version: 3 })], new Set(["h-off"]));
    expect(item?.attention).toBe(AutomationAttention.HostOffline);
  });

  it("sorts failing, then needs attention, then on, then off, each by name", () => {
    const items = automationListItems(
      [automation("d", { enabled: false }), automation("c"), automation("b", { pending_version: 1 }), automation("a2", { last_run: failedRun }), automation("a1")],
      new Set(),
    );
    expect(items.map((item) => item.automation.name)).toEqual(["a2", "b", "a1", "c", "d"]);
  });
});

describe("automationGroups", () => {
  it("leads with Custom when it holds the only failing automation", () => {
    const items = automationListItems([automation("healthy"), automation("broken", { kind: AutomationKind.Custom, last_run: failedRun })], new Set());
    expect(automationGroups(items).map((group) => group.label)).toEqual(["Custom", "Default"]);
  });

  it("keeps Default first on a tie and leaves an empty group out", () => {
    const items = automationListItems([automation("a", { kind: AutomationKind.Custom }), automation("b")], new Set());
    expect(automationGroups(items).map((group) => group.label)).toEqual(["Default", "Custom"]);
    expect(automationGroups(items.slice(1)).map((group) => group.label)).toEqual(["Default"]);
  });
});

describe("automationLastRunText and automationWeekText", () => {
  it.each([
    { runs_7d: undefined, last_run: null, want: ["never ran", null] },
    { runs_7d: { total: 0, failed: 0 }, last_run: failedRun, want: ["just now", "no runs · 7d"] },
    { runs_7d: { total: 1, failed: 0 }, last_run: failedRun, want: ["just now", "1 run · 7d"] },
    { runs_7d: { total: 38, failed: 7 }, last_run: failedRun, want: ["just now", "7 of 38 failed · 7d"] },
  ])("$want", ({ runs_7d, last_run, want }) => {
    const a = automation("x", { last_run, ...(runs_7d ? { runs_7d } : {}) });
    expect([automationLastRunText(a), automationWeekText(a)]).toEqual(want);
  });
});
