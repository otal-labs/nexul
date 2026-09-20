import { describe, expect, it } from "vitest";

import { fromBase64, toBase64 } from "@/lib/collab/binary";

describe("collab base64 helpers", () => {
  it("round-trips arbitrary binary", () => {
    const bytes = new Uint8Array([0, 1, 2, 253, 254, 255, 42]);
    expect(fromBase64(toBase64(bytes))).toEqual(bytes);
  });

  it("round-trips large payloads beyond the chunk size", () => {
    const bytes = new Uint8Array(200_000);
    for (let i = 0; i < bytes.length; i++) bytes[i] = i % 251;
    expect(fromBase64(toBase64(bytes))).toEqual(bytes);
  });

  it("round-trips the empty payload", () => {
    expect(fromBase64(toBase64(new Uint8Array(0)))).toEqual(new Uint8Array(0));
  });
});
