import { failed, type ResolveErrorKind, type ResolveResult, type Resolver } from "./resolver.ts";

// An in-memory resolver for tests: a configured string is the value, a configured kind is that error, and a reference
// nobody configured is not found.
export class FakeResolver implements Resolver {
  readonly calls: string[][] = [];

  constructor(
    readonly scheme: string,
    private readonly entries: Record<string, string | { error: ResolveErrorKind }>,
  ) {}

  async resolveAll(refs: string[]): Promise<Map<string, ResolveResult>> {
    this.calls.push(refs);
    const out = new Map<string, ResolveResult>();
    for (const ref of refs) {
      const entry = this.entries[ref];
      if (typeof entry === "string") {
        out.set(ref, { ok: true, value: entry });
        continue;
      }
      out.set(ref, failed(entry?.error ?? "not_found"));
    }
    return out;
  }
}
