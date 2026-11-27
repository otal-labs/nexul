// What a computer's runner reports, as far as readiness reads it (the app's `Computer` satisfies it).
export interface ReadinessComputer {
  runner?: { connected: boolean; reconnecting?: boolean; last_seen: string } | undefined;
  facts?: { t3: { state: string; install?: string | undefined } } | undefined;
  pair_error?: string | undefined;
  // Go's zero time while T3 Code has not paired through the runner.
  token_expires_at?: string | undefined;
}

export type Readiness = "ready" | "degraded" | "offline" | "stale";

export interface ComputerReadiness {
  state: Readiness;
  // The computer's state in the lane's own words: "Online", "Offline", or why an online one cannot take a run.
  words: string;
}

// Offline this long reads as gone for good.
export const STALE_OFFLINE_DAYS = 7;

const DAY_MS = 24 * 60 * 60 * 1000;

const stalePast = (lastSeen: string, now: number) => {
  const seen = Date.parse(lastSeen);
  return Number.isFinite(seen) && Math.floor((now - seen) / DAY_MS) >= STALE_OFFLINE_DAYS;
};

// Why T3 Code cannot take a run on a connected computer, or undefined when it can. `presence` is the live answer from the
// socket ("connected", "rejected"); without it the runner's last report of T3 Code decides.
const blockedBy = (computer: ReadinessComputer, presence: string | undefined): string | undefined => {
  const t3 = computer.facts?.t3;
  if (presence === "rejected") return "T3 Code refused the session";
  if (computer.pair_error) return "Pairing failed";
  if (presence === "connected") return undefined;
  if (presence === undefined && t3?.state === "answering") return undefined;
  if (t3?.state === "installing") return "Installing T3 Code";
  if (t3?.state === "not_running" && t3.install === "desktop_app") return "Open T3 Code";
  if (t3?.state === "not_running") return "T3 Code not running";
  if (t3?.state === "missing") return "T3 Code not installed";
  if (t3?.state === "not_loopback") return "T3 Code not on loopback";
  if (!(Date.parse(computer.token_expires_at ?? "") > 0)) return "Pairing";
  return "T3 Code not answering";
};

// One readiness for a computer on every surface: ready (can take a run), degraded (online, but T3 Code or its pairing
// blocks runs), offline, or stale (offline for a week or more). Pure; `now` is injectable for tests.
export const computerReadiness = (computer: ReadinessComputer, presence?: string, now: number = Date.now()): ComputerReadiness => {
  const runner = computer.runner;
  if (!runner) return { state: "offline", words: "Nexul app not installed" };
  if (runner.reconnecting) return { state: "degraded", words: "Reconnecting" };
  if (!runner.connected) return { state: stalePast(runner.last_seen, now) ? "stale" : "offline", words: "Offline" };
  const blocked = blockedBy(computer, presence);
  if (blocked) return { state: "degraded", words: blocked };
  return { state: "ready", words: "Online" };
};
