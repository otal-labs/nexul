import { ApiClient } from "./api-client.ts";
import type { ConfigSchema, ConfigValues } from "./config-schema.ts";

// PlayNames is filled by `nexul types` (src/plays.generated.d.ts); while empty, any label type-checks.
export interface PlayNames {}

type KnownPlays = [keyof PlayNames] extends [never] ? Record<string, "ticket"> : PlayNames;

// TicketPlay is a ticket play's label, so once names are generated a typo, a rename, or a doc play fails the check.
export type TicketPlay = { [K in keyof KnownPlays]: KnownPlays[K] extends "ticket" ? K : never }[keyof KnownPlays] & string;

export interface RunPlayOptions {
  // Whose computer the run lands on; the ticket's developer by default.
  runOn?: "developer" | "tester";
  priority?: "high" | "normal" | "low";
}

// QueuedRun is the queued run, or one that didn't run; reason is why, or what it waits on ("paused" at the daily cap).
export interface QueuedRun {
  id: string;
  state: "queued" | "didnt_run";
  reason: string;
}

// Ctx is what every handler receives: the typed API client,
// config values keyed by the schema the automation declared, workspace
// secrets by name, a logger whose output becomes the run's captured
// logs — this is the one sanctioned way to produce run output, so a
// handler never needs bare console.log to show up in run history — and
// runPlay, which queues one of the workspace's ticket plays by its label.
export interface Ctx<S extends ConfigSchema = ConfigSchema> {
  api: ApiClient;
  config: ConfigValues<S>;
  secrets: Record<string, string>;
  log: (message: string, meta?: Record<string, unknown>) => void;
  runPlay: (play: TicketPlay, ticketId: string, opts?: RunPlayOptions) => Promise<QueuedRun>;
}

export function buildCtx<S extends ConfigSchema>(
  api: ApiClient,
  config: ConfigValues<S>,
  secrets: Record<string, string>,
  log: (message: string, meta?: Record<string, unknown>) => void,
): Ctx<S> {
  const runPlay = (play: TicketPlay, ticketId: string, opts: RunPlayOptions = {}) =>
    api.request<QueuedRun>("POST", "/api/plays/queue", {
      play,
      ticket_id: ticketId,
      run_on: opts.runOn,
      priority: opts.priority,
    });
  return { api, config, secrets, log, runPlay };
}
