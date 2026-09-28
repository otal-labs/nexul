export interface RunningJob {
  id: string;
  kind: string;
  service?: string;
}

// A machine's connected runner. Enrollment and removal stay web-only; the phone reads this list to answer
// "did my deploy actually pick up a runner", nothing more.
export interface Runner {
  id: string;
  name: string;
  connected: boolean;
  last_seen: string;
  version: string;
  // The machine this runner belongs to; empty until its first connect resolves one.
  machine?: string;
  running_job: RunningJob | null;
}

// Disconnected stays neutral, not destructive: going offline isn't itself an error (mirrors the web badge).
export const runnerStatusDot = (connected: boolean): string => (connected ? "bg-success" : "bg-muted-foreground");
