import { describe, expect, test } from "bun:test";
import { FakeResolver } from "../src/fake-resolver.ts";
import { ResolverRegistry } from "../src/resolver.ts";

describe("ResolverRegistry", () => {
  test("a mixed batch returns each value or typed error, one batch per resolver", async () => {
    const op = new FakeResolver("op", {
      "op://acme/db/password": "hunter2",
      "op://acme/db/missing": { error: "field_missing" },
      "op://other/db/password": { error: "no_access" },
      "op://acme/down/password": { error: "unreachable" },
    });
    const registry = new ResolverRegistry([op]);

    const got = await registry.resolveAll([
      "op://acme/db/password",
      "op://acme/db/missing",
      "op://other/db/password",
      "op://acme/down/password",
      "op://acme/gone/password",
      "vault://x/y/z",
    ]);

    expect(op.calls).toEqual([["op://acme/db/password", "op://acme/db/missing", "op://other/db/password", "op://acme/down/password", "op://acme/gone/password"]]);
    expect(Object.fromEntries(got)).toEqual({
      "op://acme/db/password": { ok: true, value: "hunter2" },
      "op://acme/db/missing": { ok: false, error: "field_missing" },
      "op://other/db/password": { ok: false, error: "no_access" },
      "op://acme/down/password": { ok: false, error: "unreachable" },
      "op://acme/gone/password": { ok: false, error: "not_found" },
      "vault://x/y/z": { ok: false, error: "other" },
    });
  });
});
