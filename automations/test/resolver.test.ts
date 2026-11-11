import { describe, expect, test } from "bun:test";
import { ResolverRegistry } from "../src/resolver.ts";
import { FakeResolver } from "./fake-resolver.ts";

describe("ResolverRegistry", () => {
  test("sends a resolver one batch of all its references and answers a reference no resolver serves as other", async () => {
    const op = new FakeResolver("op", { "op://acme/db/password": "hunter2", "op://acme/api/key": "k1" });
    const registry = new ResolverRegistry([op]);

    const got = await registry.resolveAll(["op://acme/db/password", "literal", "op://acme/api/key", "nope://x/y/z"]);

    expect(op.calls).toEqual([["op://acme/db/password", "op://acme/api/key"]]);
    expect(Object.fromEntries(got)).toEqual({
      "op://acme/db/password": { ok: true, value: "hunter2" },
      "op://acme/api/key": { ok: true, value: "k1" },
      literal: { ok: false, error: "other" },
      "nope://x/y/z": { ok: false, error: "other" },
    });
  });
});
