import { describe, expect, it } from "vitest";

import { computerReadiness, type ReadinessComputer } from "@nexul/client-core/computerReadiness";

const NOW = Date.parse("2026-11-27T12:00:00Z");
const paired = "2099-01-01T00:00:00Z";
const online = { connected: true, last_seen: "2026-11-27T11:59:00Z" };

const cases: [string, ReadinessComputer, string | undefined, string, string][] = [
  ["connected, T3 Code answering by presence", { runner: online, token_expires_at: paired }, "connected", "ready", "Online"],
  ["connected, T3 Code answering by its report when no presence is known", { runner: online, token_expires_at: paired, facts: { t3: { state: "answering" } } }, undefined, "ready", "Online"],
  ["connected, presence says it is not answering", { runner: online, token_expires_at: paired, facts: { t3: { state: "answering" } } }, "", "degraded", "T3 Code not answering"],
  ["connected, T3 Code's desktop app closed", { runner: online, token_expires_at: paired, facts: { t3: { state: "not_running", install: "desktop_app" } } }, "", "degraded", "Open T3 Code"],
  ["connected, T3 Code refused the session", { runner: online, token_expires_at: paired }, "rejected", "degraded", "T3 Code refused the session"],
  ["connected, pairing failed", { runner: online, token_expires_at: paired, pair_error: "no token" }, "", "degraded", "Pairing failed"],
  ["connected, never paired", { runner: online, token_expires_at: "0001-01-01T00:00:00Z" }, "", "degraded", "Pairing"],
  ["dropped a moment ago", { runner: { connected: false, reconnecting: true, last_seen: "2026-11-27T11:59:50Z" } }, undefined, "degraded", "Reconnecting"],
  ["offline for hours", { runner: { connected: false, last_seen: "2026-11-27T09:00:00Z" } }, undefined, "offline", "Offline"],
  ["offline for six days", { runner: { connected: false, last_seen: "2026-11-21T12:00:00Z" } }, undefined, "offline", "Offline"],
  ["offline for seven days", { runner: { connected: false, last_seen: "2026-11-20T12:00:00Z" } }, undefined, "stale", "Offline"],
  ["offline for fifteen days", { runner: { connected: false, last_seen: "2026-11-12T12:00:00Z" } }, undefined, "stale", "Offline"],
  ["no Nexul app yet", {}, undefined, "offline", "Nexul app not installed"],
];

describe("computerReadiness", () => {
  it.each(cases)("%s", (_, computer, presence, state, words) => {
    expect(computerReadiness(computer, presence, NOW)).toEqual({ state, words });
  });
});
