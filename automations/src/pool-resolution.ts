import { log } from "./log.ts";
import type { ResolveErrorKind, ResolveResult } from "./resolver.ts";
import { parseSecretReference } from "./secret-reference.ts";

// What a failed entry reads in the log; a reference or a value never rides in it.
const REASONS: Record<ResolveErrorKind, string> = {
  not_found: "not found",
  field_missing: "field not found",
  no_access: "no access",
  unreachable: "1Password unreachable",
  no_token: "no 1Password token for this workspace on the host",
  other: "could not be resolved",
};

// A pool holds a few dozen names at most; a thread asking for more is not an automation.
const MAX_ENTRIES = 256;

export interface Asker {
  workspaceId: string;
  automationId: string;
  automationName: string;
}

export interface PoolResolver {
  // Pool entries by name to the values of the ones that resolved; a failed name is absent.
  resolve(asker: Asker, entries: unknown): Promise<Record<string, string>>;
}

interface Batch {
  refs: Set<string>;
  done: Promise<Map<string, ResolveResult>>;
}

// Answers a worker thread's request for the references in its hello. The thread says what to resolve and never whom
// it is: the workspace is the one the supervisor assigned it, so a request can only ever use that workspace's token
// (ADR 0182). Requests of one workspace arriving within windowMs share one resolve, so a host restart costs one read
// per workspace.
export class PoolResolution implements PoolResolver {
  private readonly batches = new Map<string, Batch>();

  constructor(
    private readonly managers: { resolve(workspaceId: string, refs: string[]): Promise<Map<string, ResolveResult>> },
    private readonly windowMs = 1000,
  ) {}

  async resolve(asker: Asker, entries: unknown): Promise<Record<string, string>> {
    const asked = Object.entries(isRecord(entries) ? entries : {})
      .filter((entry): entry is [string, string] => typeof entry[1] === "string" && parseSecretReference(entry[1]).kind === "reference")
      .slice(0, MAX_ENTRIES);
    if (asked.length === 0) return {};

    const results = await this.batch(
      asker.workspaceId,
      asked.map(([, ref]) => ref),
    );
    const values: Record<string, string> = {};
    for (const [name, ref] of asked) {
      const result = results.get(ref);
      if (result?.ok) {
        values[name] = result.value;
        continue;
      }
      log("warn", `${name}: ${REASONS[result?.error ?? "other"]}`, { automationId: asker.automationId, automationName: asker.automationName });
    }
    return values;
  }

  private batch(workspaceId: string, refs: string[]): Promise<Map<string, ResolveResult>> {
    let batch = this.batches.get(workspaceId);
    if (!batch) {
      const pending = new Set<string>();
      const done = new Promise<Map<string, ResolveResult>>((resolve, reject) => {
        setTimeout(() => {
          this.batches.delete(workspaceId);
          this.managers.resolve(workspaceId, [...pending]).then(resolve, reject);
        }, this.windowMs);
      });
      batch = { refs: pending, done };
      this.batches.set(workspaceId, batch);
    }
    for (const ref of refs) batch.refs.add(ref);
    return batch.done;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
