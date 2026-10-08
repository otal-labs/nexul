import type { Deploy, DeployLogLine, DeployPhase } from "@/models/Stack";

export type DeployStepState = "pending" | "active" | "done" | "failed" | "skipped";

export interface DeployStep {
  key: "wait" | DeployPhase;
  label: string;
  state: DeployStepState;
  durationMs: number | undefined;
}

export interface DeployProgress {
  title: string;
  steps: DeployStep[];
}

type Phase = Exclude<DeployPhase, "">;

const phaseLabels: Record<Phase, string> = {
  checkout: "Cloning repository",
  build: "Building",
  deploy: "Deploying",
};

const phasesFor = (deploy: Deploy): Phase[] => (deploy.kind === "build" ? ["checkout", "build", "deploy"] : ["deploy"]);

export const isTerminal = (deploy: Deploy): boolean => deploy.status === "healthy" || deploy.status === "failed";

const titleFor = (deploy: Deploy): string => {
  if (deploy.status === "failed") return "Deploy failed";
  if (deploy.status === "healthy") return "Deployed";
  if (deploy.kind === "build") return "Building and deploying";
  return "Deploying";
};

const lastStepStateFor = (deploy: Deploy): DeployStepState => {
  if (deploy.status === "failed") return "failed";
  if (deploy.status === "healthy") return "done";
  return "active";
};

// Steps are derived, not stored: the runner only emits phase-tagged lines, so a phase starts at its first line
// and ends where the next started phase begins (or at updated_at once terminal, else `now` so it ticks).
export const deriveDeployProgress = (deploy: Deploy, lines: DeployLogLine[], now: number): DeployProgress => {
  const terminal = isTerminal(deploy);
  const phases = phasesFor(deploy);
  const firstTs = new Map<Phase, number>();
  for (const line of lines) {
    if (line.phase !== "" && !firstTs.has(line.phase)) firstTs.set(line.phase, line.ts);
  }
  const lastStartedIndex = phases.reduce((last, phase, i) => (firstTs.has(phase) ? i : last), -1);
  const endAt = terminal ? Date.parse(deploy.updated_at) : now;
  const lastState = lastStepStateFor(deploy);

  // Waiting ends at the first line of any kind, so a phase-less "deploy failed: …" still closes it.
  const waitStart = Date.parse(deploy.created_at);
  const waitEnd = lines[0]?.ts ?? endAt;
  const steps: DeployStep[] = [
    {
      key: "wait",
      label: "Waiting for a runner",
      state: lastStartedIndex === -1 ? lastState : "done",
      durationMs: Math.max(0, waitEnd - waitStart),
    },
  ];

  phases.forEach((phase, i) => {
    const start = firstTs.get(phase);
    if (start === undefined) {
      const skipped = i < lastStartedIndex || terminal;
      steps.push({ key: phase, label: phaseLabels[phase], state: skipped ? "skipped" : "pending", durationMs: undefined });
      return;
    }
    const nextStart = phases
      .slice(i + 1)
      .map((p) => firstTs.get(p))
      .find((t) => t !== undefined);
    steps.push({
      key: phase,
      label: phaseLabels[phase],
      state: nextStart === undefined ? lastState : "done",
      durationMs: Math.max(0, (nextStart ?? endAt) - start),
    });
  });

  return { title: titleFor(deploy), steps };
};

export const formatStepDuration = (ms: number | undefined): string => {
  if (ms === undefined) return "—";
  const total = Math.floor(ms / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
};

const pad = (n: number, width = 2): string => String(n).padStart(width, "0");

// Local wall-clock time at millisecond precision, matching what the operator sees on the runner host.
export const formatLogTimestamp = (ts: number): string => {
  const d = new Date(ts);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${pad(d.getMilliseconds(), 3)}`;
};

export const logToText = (lines: DeployLogLine[]): string =>
  lines.map((line) => `${formatLogTimestamp(line.ts)}  ${line.text}`).join("\n");
