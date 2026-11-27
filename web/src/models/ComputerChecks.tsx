import type { SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { reachedThroughRunner } from "@/models/ComputerFacts";
import { stillPairing, type Computer, type ComputerRunner } from "@/models/Pairing";
import { daysAgo, formatRelativeTime, formatShortDate } from "@/utils/TimeUtility";

export type CheckState = "waiting" | "passed" | "warning" | "failed";

// One live check a dialog waits on: its name, a mono line on what it found, and its state in a word.
export interface Check {
  name: string;
  detail: string;
  state: CheckState;
  label: string;
}

// Added with Add a computer, or moved to its app: reached through a runner, or with no tunnel and no address yet.
export const addedWithRunner = (computer: Computer) =>
  reachedThroughRunner(computer.server_url) || (!computer.tunnel && computer.server_url === "");

// Paired through the computer's own app; a tunnel computer being moved still holds its tunnel session until then.
export const pairedThroughRunner = (computer: Computer) => reachedThroughRunner(computer.server_url) && !stillPairing(computer);

const runnerConnected = (computer: Computer) => computer.runner?.connected === true;

const t3Answering = (computer: Computer) => runnerConnected(computer) && computer.facts?.t3.state === "answering";

const connectedCheck = (computer: Computer): Check => {
  const base = { name: "Computer connected" };
  if (!computer.runner) return { ...base, detail: "Run the command on the computer", state: "waiting", label: "Waiting" };
  if (computer.runner.reconnecting) return { ...base, detail: "Its connection dropped; waiting for it to come back", state: "waiting", label: "Reconnecting" };
  if (!computer.runner.connected) return { ...base, detail: "It went offline · journalctl -u nexul-computer", state: "warning", label: "Offline" };
  const { hostname, os, arch } = computer.facts ?? {};
  const detail = [hostname, [os, arch].filter(Boolean).join(" ")].filter(Boolean).join(" · ");
  return { ...base, detail: detail || "The Nexul app is running", state: "passed", label: "Connected" };
};

const t3Check = (computer: Computer): Check => {
  const base = { name: "T3 Code found" };
  const t3 = computer.facts?.t3;
  if (!runnerConnected(computer)) return { ...base, detail: "Checked once the computer connects", state: "waiting", label: "Waiting" };
  if (!t3) return { ...base, detail: "Looking for T3 Code", state: "waiting", label: "Looking" };
  if (t3.state === "answering") {
    const detail = [`T3 Code ${t3.version ?? ""}`.trim(), t3.port && `port ${t3.port}`].filter(Boolean).join(" · ");
    return { ...base, detail, state: "passed", label: "Found" };
  }
  if (t3.state === "installing") return { ...base, detail: "The command is installing T3 Code", state: "waiting", label: "Installing" };
  if (t3.state === "not_running" && t3.install === "desktop_app") return { ...base, detail: "Open T3 Code on the computer", state: "warning", label: "Closed" };
  if (t3.state === "not_running") return { ...base, detail: t3.restart_error ?? "Its background service isn't answering", state: "warning", label: "Not running" };
  if (t3.state === "missing") return { ...base, detail: "Not installed; run the command again without --no-t3", state: "failed", label: "Missing" };
  return { ...base, detail: "It listens beyond this computer; serve it on 127.0.0.1", state: "failed", label: "Not local" };
};

const pairedCheck = (computer: Computer): Check => {
  const base = { name: "Paired" };
  if (pairedThroughRunner(computer)) return { ...base, detail: `Session until ${formatShortDate(computer.token_expires_at)}`, state: "passed", label: "Paired" };
  if (computer.pair_error) return { ...base, detail: computer.pair_error, state: "failed", label: "Failed" };
  if (t3Answering(computer)) return { ...base, detail: "Pairing through the computer's connection", state: "waiting", label: "Pairing" };
  return { ...base, detail: "Pairs once T3 Code is found", state: "waiting", label: "Waiting" };
};

// The three checks Add a computer waits on, each driven by what the computer list says after a pushed frame.
export const computerChecks = (computer: Computer): Check[] => [connectedCheck(computer), t3Check(computer), pairedCheck(computer)];

// A computer unseen this long is likely gone for good, so its pane offers Remove beside the warning.
export const STALE_OFFLINE_DAYS = 7;

const staleSince = (lastSeen: string) => daysAgo(lastSeen) >= STALE_OFFLINE_DAYS;

// "3h ago", "15d ago": short units, so a state line never wraps.
export const seenAgo = (lastSeen: string) => formatRelativeTime(lastSeen);

export interface Lane {
  tone: SettingsStatusTone;
  text: string;
  detail?: string;
  // Offline reads as a ring dot on every surface.
  hollow?: boolean;
}

const t3Lane = (computer: Computer, presence: string | undefined): Lane => {
  const t3 = computer.facts?.t3;
  if (presence === "connected") return { tone: "success", text: "T3 Code answering" };
  if (presence === "rejected") return { tone: "destructive", text: "T3 Code refused the session", detail: "re-pair to connect again" };
  if (computer.pair_error) return { tone: "destructive", text: "Pairing failed", detail: computer.pair_error };
  if (t3?.state === "installing") return { tone: "info", text: "Installing T3 Code" };
  if (t3?.state === "not_running" && t3.install === "desktop_app") return { tone: "warning", text: "Open T3 Code" };
  if (t3?.state === "not_running") return { tone: "warning", text: "T3 Code not running" };
  if (t3?.state === "missing") return { tone: "destructive", text: "T3 Code not installed" };
  if (t3?.state === "not_loopback") return { tone: "destructive", text: "T3 Code not on loopback" };
  if (stillPairing(computer)) return { tone: "warning", text: "Pairing" };
  return { tone: "warning", text: "T3 Code not answering" };
};

// A runner computer's health, one lane at a time: the computer's own connection, then T3 Code through it.
export const computerLanes = (computer: Computer, presence: string | undefined, repairing = false): Lane[] => {
  if (!computer.runner) return [{ tone: "muted", text: "Nexul app not installed" }];
  if (computer.runner.reconnecting) return [{ tone: "warning", text: "Reconnecting", detail: `dropped ${formatRelativeTime(computer.runner.last_seen)}` }];
  if (!computer.runner.connected) return [{ ...runnerPresence(computer.runner), detail: `seen ${seenAgo(computer.runner.last_seen)}` }];
  if (repairing) return [{ tone: "success", text: "Online" }, { tone: "info", text: "Re-pairing" }];
  return [{ tone: "success", text: "Online" }, t3Lane(computer, presence)];
};

const toneRank: Record<SettingsStatusTone, number> = { destructive: 0, warning: 1, info: 2, muted: 3, success: 4 };

// A row.s states, the one that needs doing first.
export const worstFirst = <T extends { tone: SettingsStatusTone }>(lanes: T[]): T[] => [...lanes].sort((a, b) => toneRank[a.tone] - toneRank[b.tone]);

// A computer's runner in a word: online, back any moment after a drop, or offline.
// One mapping for every surface: online `success`, offline grey, offline for a week or more `warning`.
export const runnerPresence = (runner: ComputerRunner | undefined): Lane => {
  if (runner?.connected) return { tone: "success", text: "Online" };
  if (runner?.reconnecting) return { tone: "warning", text: "Reconnecting" };
  if (runner && staleSince(runner.last_seen)) return { tone: "warning", text: "Offline", hollow: true };
  return { tone: "muted", text: "Offline", hollow: true };
};

export const offlineForDays = (computer: Computer): number => {
  const runner = computer.runner;
  if (!runner || runner.connected || runner.reconnecting) return 0;
  const days = daysAgo(runner.last_seen);
  return Number.isNaN(days) ? 0 : days;
};

export const isStaleOffline = (computer: Computer) => offlineForDays(computer) >= STALE_OFFLINE_DAYS;

export interface StateLine {
  tone: SettingsStatusTone;
  text: string;
  // The part of the line that is about how long it has been gone; it alone takes the warning colour.
  seen?: string;
  hollow?: boolean;
}

// One short line for a list row: the lane that needs doing first, "Online · idle" or "Online · 2 of 4 running" when all is well, "seen 16m ago" when offline.
export const computerStateLine = (computer: Computer, presence: string | undefined): StateLine => {
  if (computer.shared) return runnerPresence(computer.runner);
  const [lane] = worstFirst(computerLanes(computer, presence));
  if (!lane) return { tone: "muted", text: "" };
  if (lane.text === "Online" || lane.text === "T3 Code answering") return { tone: "success", text: `Online · ${computer.running > 0 ? `${computer.running} of ${computer.runs_at_once} running` : "idle"}` };
  if (lane.text === "Offline" && computer.runner) return { tone: lane.tone, text: "Offline · ", seen: `seen ${seenAgo(computer.runner.last_seen)}`, hollow: true };
  return { tone: lane.tone, text: lane.text };
};

// The muted words after a computer's name in a ranked list: nothing while online, "seen 3h ago" once offline.
export const offlineNote = (computer: Computer): string | undefined => {
  if (!computer.runner || computer.runner.connected || computer.runner.reconnecting) return undefined;
  return `seen ${seenAgo(computer.runner.last_seen)}`;
};
