import { parseSecretReference } from "./secret-reference.ts";

export type ResolveErrorKind = "not_found" | "field_missing" | "no_access" | "unreachable" | "other";

// A failure carries only its kind: a reference or a value never rides in an error.
export type ResolveResult = { ok: true; value: string } | { ok: false; error: ResolveErrorKind };

// Whether a token works: its client opened, the manager refused the token, or the manager could not be reached.
export type ManagerState = "ready" | "token rejected" | "unreachable";

export interface Resolver {
  readonly scheme: string;
  // Opens the manager's client with the token and reports how that went; it reads no secret and does not throw.
  check(): Promise<ManagerState>;
  // Returns one result per reference given; it does not throw for a failed reference or an unreachable manager.
  resolveAll(refs: string[]): Promise<Map<string, ResolveResult>>;
}

export const failed = (error: ResolveErrorKind): ResolveResult => ({ ok: false, error });

export class ResolverRegistry {
  private readonly byScheme = new Map<string, Resolver>();

  constructor(resolvers: Resolver[]) {
    for (const r of resolvers) this.byScheme.set(r.scheme, r);
  }

  // Groups references by scheme, one batch per resolver; a reference no resolver serves comes back as "other".
  async resolveAll(refs: string[]): Promise<Map<string, ResolveResult>> {
    const out = new Map<string, ResolveResult>();
    const groups = new Map<Resolver, string[]>();
    for (const ref of refs) {
      const scheme = parseSecretReference(ref).ref?.scheme;
      const resolver = scheme === undefined ? undefined : this.byScheme.get(scheme);
      if (!resolver) {
        out.set(ref, failed("other"));
        continue;
      }
      groups.set(resolver, [...(groups.get(resolver) ?? []), ref]);
    }
    const batches = await Promise.all([...groups].map(([resolver, group]) => resolver.resolveAll(group)));
    for (const batch of batches) for (const [ref, result] of batch) out.set(ref, result);
    return out;
  }
}
