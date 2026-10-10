import type { SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { reachedThroughRunner } from "@/models/ComputerFacts";
import { stillPairing, type Computer } from "@/models/Pairing";
import { formatRelativeTime, formatShortDate } from "@/utils/TimeUtility";

export type CheckState = "waiting" | "passed" | "warning" | "failed";

// One live check a dialog waits on: its name, a mono line on what it found, and its state in a word.
export interface Check {
  name: string;
  detail: string;
  state: CheckState;
  label: string;
}

// Added with Add a computer: no tunnel, and reached through a runner once it pairs.
export const addedWithRunner = (computer: Computer) =>
  !computer.tunnel && (!!computer.runner || computer.server_url === "" || reachedThroughRunner(computer.server_url));

const runnerConnected = (computer: Computer) => computer.runner?.connected === true;

const t3Answering = (computer: Computer) => runnerConnected(computer) && computer.facts?.t3.state === "answering";

const connectedCheck = (computer: Computer): Check => {
  const base = { name: "Computer connected" };
  if (!computer.runner) return { ...base, detail: "Run the command on the computer", state: "waiting", label: "Waiting" };
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
  if (t3.state === "not_running" && t3.install === "desktop_app") return { ...base, detail: "Open T3 Code on the computer", state: "warning", label: "Closed" };
  if (t3.state === "not_running") return { ...base, detail: t3.restart_error ?? "Its background service isn't answering", state: "warning", label: "Not running" };
  if (t3.state === "missing") return { ...base, detail: "Not installed; run the command again without --no-t3", state: "failed", label: "Missing" };
  return { ...base, detail: "It listens beyond this computer; serve it on 127.0.0.1", state: "failed", label: "Not local" };
};

const pairedCheck = (computer: Computer): Check => {
  const base = { name: "Paired" };
  if (!stillPairing(computer)) return { ...base, detail: `Session until ${formatShortDate(computer.token_expires_at)}`, state: "passed", label: "Paired" };
  if (computer.pair_error) return { ...base, detail: computer.pair_error, state: "failed", label: "Failed" };
  if (t3Answering(computer)) return { ...base, detail: "Pairing through the computer's connection", state: "waiting", label: "Pairing" };
  return { ...base, detail: "Pairs once T3 Code is found", state: "waiting", label: "Waiting" };
};

// The three checks Add a computer waits on, each driven by what the computer list says after a pushed frame.
export const computerChecks = (computer: Computer): Check[] => [connectedCheck(computer), t3Check(computer), pairedCheck(computer)];

export interface Lane {
  tone: SettingsStatusTone;
  text: string;
  detail?: string;
}

const t3Lane = (computer: Computer, presence: string | undefined): Lane => {
  const t3 = computer.facts?.t3;
  if (presence === "connected") return { tone: "success", text: "T3 Code answering" };
  if (computer.pair_error) return { tone: "destructive", text: "Pairing failed", detail: computer.pair_error };
  if (t3?.state === "not_running" && t3.install === "desktop_app") return { tone: "warning", text: "Open T3 Code" };
  if (t3?.state === "not_running") return { tone: "warning", text: "T3 Code not running" };
  if (t3?.state === "missing") return { tone: "destructive", text: "T3 Code not installed" };
  if (t3?.state === "not_loopback") return { tone: "destructive", text: "T3 Code not on loopback" };
  if (stillPairing(computer)) return { tone: "warning", text: "Pairing" };
  return { tone: "muted", text: "T3 Code not answering" };
};

// A runner computer's health, one lane at a time: the computer's own connection, then T3 Code through it.
export const computerLanes = (computer: Computer, presence: string | undefined): Lane[] => {
  if (!computer.runner) return [{ tone: "muted", text: "Nexul app not installed" }];
  if (!computer.runner.connected) return [{ tone: "muted", text: "Offline", detail: `last seen ${formatRelativeTime(computer.runner.last_seen)}` }];
  return [{ tone: "success", text: "Online" }, t3Lane(computer, presence)];
};
