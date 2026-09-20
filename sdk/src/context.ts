import { ApiClient } from "./api-client.ts";
import type { ConfigSchema, ConfigValues } from "./config-schema.ts";

// Ctx is what every handler receives: the typed API client,
// config values keyed by the schema the automation declared, workspace
// secrets by name, and a logger whose output becomes the run's captured
// logs — this is the one sanctioned way to produce run output, so a
// handler never needs bare console.log to show up in run history.
export interface Ctx<S extends ConfigSchema = ConfigSchema> {
  api: ApiClient;
  config: ConfigValues<S>;
  secrets: Record<string, string>;
  log: (message: string, meta?: Record<string, unknown>) => void;
}

export function buildCtx<S extends ConfigSchema>(
  api: ApiClient,
  config: ConfigValues<S>,
  secrets: Record<string, string>,
  log: (message: string, meta?: Record<string, unknown>) => void,
): Ctx<S> {
  return { api, config, secrets, log };
}
