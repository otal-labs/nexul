import { describe, expect, test } from "bun:test";
import type { Client } from "@1password/sdk";
import { OnePasswordResolver } from "../src/onepassword-resolver.ts";

type Responses = Awaited<ReturnType<Client["secrets"]["resolveAll"]>>["individualResponses"];

function clientAnswering(individualResponses: Responses, reads: string[][]) {
  return { secrets: { resolveAll: async (refs: string[]) => (reads.push(refs), { individualResponses }) } } as unknown as Pick<Client, "secrets">;
}

describe("OnePasswordResolver", () => {
  test("maps each per-reference error, keeps the good entries, and reads once through one client", async () => {
    const reads: string[][] = [];
    let opened = 0;
    const resolver = new OnePasswordResolver("token", async () => {
      opened++;
      return clientAnswering(
        {
          "op://a/i/f": { content: { secret: "s3cret", itemId: "i", vaultId: "v" } },
          "op://a/none/f": { error: { type: "itemNotFound" } },
          "op://a/i/none": { error: { type: "fieldNotFound" } },
          "op://hidden/i/f": { error: { type: "vaultNotFound" } },
          "op://a/dup/f": { error: { type: "tooManyItems" } },
        },
        reads,
      );
    });
    const refs = ["op://a/i/f", "op://a/none/f", "op://a/i/none", "op://hidden/i/f", "op://a/dup/f"];

    const got = await resolver.resolveAll(refs);
    await resolver.resolveAll(refs);

    expect(Object.fromEntries(got)).toEqual({
      "op://a/i/f": { ok: true, value: "s3cret" },
      "op://a/none/f": { ok: false, error: "not_found" },
      "op://a/i/none": { ok: false, error: "field_missing" },
      "op://hidden/i/f": { ok: false, error: "no_access" },
      "op://a/dup/f": { ok: false, error: "other" },
    });
    expect(reads).toEqual([refs, refs]);
    expect(opened).toBe(1);
  });

  test("a client that cannot be created fails every reference as unreachable, then is tried again", async () => {
    let attempts = 0;
    const reads: string[][] = [];
    const resolver = new OnePasswordResolver("token", async () => {
      attempts++;
      if (attempts === 1) throw new Error("dial tcp my.1password.com: no such host");
      return clientAnswering({ "op://a/i/f": { content: { secret: "s3cret", itemId: "i", vaultId: "v" } } }, reads);
    });

    const down = await resolver.resolveAll(["op://a/i/f", "op://a/i/g"]);
    const up = await resolver.resolveAll(["op://a/i/f"]);

    expect(Object.fromEntries(down)).toEqual({ "op://a/i/f": { ok: false, error: "unreachable" }, "op://a/i/g": { ok: false, error: "unreachable" } });
    expect(up.get("op://a/i/f")).toEqual({ ok: true, value: "s3cret" });
  });
});
