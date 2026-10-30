import type { Machine, OpenEnrollment, QueuedJob, Runner } from "@/models/Runner";
import { formatRelativeTime } from "@/utils/TimeUtility";

/** One machine's band in the fleet list, or the last group (machine undefined) for what no machine holds yet. */
export interface FleetGroup {
  machine?: Machine;
  runners: Runner[];
  codes: OpenEnrollment[];
}

// What no machine holds (a runner from before machines existed, a code for no machine) goes in one last group.
export const groupFleet = (machines: Machine[], runners: Runner[], codes: OpenEnrollment[]): FleetGroup[] => {
  const known = new Set(machines.map((m) => m.name));
  const groups: FleetGroup[] = machines.map((machine) => ({
    machine,
    runners: runners.filter((r) => r.machine === machine.name),
    codes: codes.filter((c) => c.machine === machine.name),
  }));
  const loose = {
    runners: runners.filter((r) => !r.machine || !known.has(r.machine)),
    codes: codes.filter((c) => !c.machine || !known.has(c.machine)),
  };
  if (loose.runners.length + loose.codes.length > 0) groups.push(loose);
  return groups;
};

export type QueueReason = "no runner online" | "all runners busy" | "next in line";

// Why a job waits, read from the runners already on the page: a job for no machine in particular fits any runner.
export const queueReason = (job: Pick<QueuedJob, "target">, runners: Runner[]): QueueReason => {
  const pool = runners.filter((r) => r.connected && (!job.target || r.machine === job.target));
  if (pool.length === 0) return "no runner online";
  if (pool.every((r) => r.running_job)) return "all runners busy";
  return "next in line";
};

export interface FleetCounts {
  total: number;
  online: number;
  busy: number;
  offline: number;
}

export const fleetCounts = (runners: Runner[]): FleetCounts => {
  const online = runners.filter((r) => r.connected).length;
  return { total: runners.length, online, busy: runners.filter((r) => r.running_job).length, offline: runners.length - online };
};

// The last dash segments (10 characters at most) stay whole while the head truncates: "…node-14" vs "…node-15".
export const splitTail = (name: string): [head: string, tail: string] => {
  let cut = -1;
  for (let at = name.lastIndexOf("-"); at > 0 && name.length - at <= 10; at = name.lastIndexOf("-", at - 1)) cut = at;
  if (cut <= 0) return [name, ""];
  return [name.slice(0, cut + 1), name.slice(cut + 1)];
};

export type GroupTone = "online" | "partial" | "offline";

// A group's dot and its state in words; unused codes are named, since they are rows but not runners.
export const groupState = (runners: Runner[], codes: number, lastSeen?: string): { tone: GroupTone; words: string } => {
  const online = runners.filter((r) => r.connected).length;
  const codeWords = codes > 0 ? ` · ${codes} unused code${codes === 1 ? "" : "s"}` : "";
  if (runners.length > 0 && online === runners.length) return { tone: "online", words: `online${codeWords}` };
  if (online > 0) return { tone: "partial", words: `${online} of ${runners.length} online${codeWords}` };
  const seen = lastSeen ? ` · seen ${formatRelativeTime(lastSeen)}` : "";
  return { tone: "offline", words: `offline${seen}${codeWords}` };
};
