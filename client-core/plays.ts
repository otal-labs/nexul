import type { TrailState } from "@nexul/client-core/trail";

// Mirrors the Go backend's five fixed board stages (ADR 0022); a ticket play shows on exactly one.
export const PLAY_STAGES = ["backlog", "progress", "review", "testing", "done"] as const;
export type PlayStage = (typeof PLAY_STAGES)[number];

export const PLAY_STAGE_LABELS: Record<PlayStage, string> = {
  backlog: "Backlog",
  progress: "In progress",
  review: "Review",
  testing: "Testing",
  done: "Done",
};

// Mirrors internal/plays.RunHealth: what the runs this person can see say about a play. last_run is null until it has run.
export interface PlayRunHealth {
  last_run: { state: TrailState; started_at: string } | null;
  runs_7d: number;
  failed_7d: number;
  waiting: number;
  running: number;
}

export const needAnswers = (n: number) => (n === 1 ? "1 needs an answer" : `${n} need answers`);

// What needs a look now, busiest first: runs in flight, then runs waiting on an answer, then a newest run that failed.
export const playAttention = (play: PlayRunHealth): { state: TrailState; text: string } | undefined => {
  if (play.running > 0) return { state: "running", text: `${play.running} running` };
  if (play.waiting > 0) return { state: "waiting", text: needAnswers(play.waiting) };
  if (play.last_run?.state === "failed") return { state: "failed", text: "Last run failed" };
  return undefined;
};

const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

export interface HealthFigure {
  state?: "running" | "waiting" | "failed";
  text: string;
}

// What the play's runs say, busiest first; each app adds "last … ago" in its own clock only while nothing runs, where it would be that run's own start.
export const healthFigures = (health: PlayRunHealth): HealthFigure[] => {
  const figures: HealthFigure[] = [];
  if (health.running > 0) figures.push({ state: "running", text: `${health.running} running` });
  if (health.waiting > 0) figures.push({ state: "waiting", text: needAnswers(health.waiting) });
  if (health.failed_7d > 0) {
    figures.push({ ...(health.last_run?.state === "failed" && { state: "failed" as const }), text: `${health.failed_7d} failed this week` });
  }
  figures.push({ text: `${plural(health.runs_7d, "run")} this week` });
  return figures;
};
