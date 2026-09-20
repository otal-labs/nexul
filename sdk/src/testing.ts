import { ApiClient, type FetchLike } from "./api-client.ts";
import type { ConfigSchema, ConfigValues } from "./config-schema.ts";
import { defaultConfigValues } from "./config-schema.ts";
import type { Ctx } from "./context.ts";
export { eventFixtures, TOPICS } from "./events.generated.ts";
export type { Topic, EventPayloads } from "./events.generated.ts";

export interface RecordedCall {
  method: string;
  path: string;
  body?: unknown;
}

export interface MockContext<S extends ConfigSchema> extends Ctx<S> {
  calls: RecordedCall[];
  logs: string[];
}

export interface MockContextOptions<S extends ConfigSchema> {
  config?: Partial<ConfigValues<S>>;
  secrets?: Record<string, string>;
  // responses maps "METHOD path" to the value request() should resolve
  // with, so a test can script "what the API would have said" without a
  // live server (dev is mock-first, no live effects).
  responses?: Record<string, unknown>;
}

// createMockContext gives handler tests (and the `dev` harness) a ctx whose
// api client records every call instead of making it — no live effects, ever.
export function createMockContext<S extends ConfigSchema>(schema: S, opts: MockContextOptions<S> = {}): MockContext<S> {
  const calls: RecordedCall[] = [];
  const logs: string[] = [];
  const fetchImpl: FetchLike = async (input, init) => {
    const path = new URL(String(input)).pathname;
    const method = init?.method ?? "GET";
    const body = init?.body ? JSON.parse(String(init.body)) : undefined;
    calls.push({ method, path, body });
    const response = opts.responses?.[`${method} ${path}`];
    return new Response(response !== undefined ? JSON.stringify(response) : null, { status: 200 });
  };
  const api = new ApiClient({ baseUrl: "http://mock.invalid", token: "mock", fetchImpl });
  const config = { ...defaultConfigValues(schema), ...opts.config } as ConfigValues<S>;
  const log = (message: string, meta?: Record<string, unknown>) => {
    logs.push(JSON.stringify({ message, ...meta }));
  };
  return { api, config, secrets: opts.secrets ?? {}, log, calls, logs };
}
