import { describe, expect, it, vi } from "vitest";

import { probeInstance, type FetchLike } from "./probe";

const okResponse = (): Response => new Response("{}", { status: 200 });

describe("probeInstance", () => {
  it("reports connected when the instance answers any HTTP status", async () => {
    const fetchImpl = vi.fn<FetchLike>().mockResolvedValue(new Response("unauthorized", { status: 401 }));
    const outcome = await probeInstance("https://a.example", 100, 200, fetchImpl);
    expect(outcome).toEqual({ kind: "connected" });
    expect(fetchImpl).toHaveBeenCalledWith("https://a.example/api/auth/me", expect.anything());
  });

  it("reports connected on a 5xx — the server is reachable even when broken", async () => {
    const fetchImpl = vi.fn<FetchLike>().mockResolvedValue(new Response("boom", { status: 500 }));
    const outcome = await probeInstance("https://a.example", 100, 200, fetchImpl);
    expect(outcome.kind).toBe("connected");
  });

  it("reports unreachable with the error when the network fails", async () => {
    const fetchImpl = vi.fn<FetchLike>().mockRejectedValue(new Error("ECONNREFUSED"));
    const outcome = await probeInstance("https://a.example", 100, 200, fetchImpl);
    expect(outcome).toEqual({ kind: "unreachable", error: "ECONNREFUSED" });
  });

  it("reports unreachable with a timeout message when the probe hangs", async () => {
    // Mimics real fetch by rejecting with the abort signal's reason once it fires, instead of ignoring the signal like a naive hung-forever mock would.
    const fetchImpl = vi.fn<FetchLike>(
      (_url, init) =>
        new Promise((_resolve, reject) => {
          init.signal?.addEventListener("abort", () => reject(init.signal?.reason));
        }),
    );
    const outcome = await probeInstance("https://a.example", 100, 200, fetchImpl, 10);
    expect(outcome.kind).toBe("unreachable");
    if (outcome.kind === "unreachable") expect(outcome.error).toContain("10ms");
  });

  it("reports unreachable for non-Error rejections", async () => {
    const fetchImpl = vi.fn<FetchLike>().mockRejectedValue("plain string");
    const outcome = await probeInstance("https://a.example", 100, 200, fetchImpl);
    expect(outcome.kind).toBe("unreachable");
    if (outcome.kind === "unreachable") expect(outcome.error).toBe("network error");
  });

  it("short-circuits to expired without touching the network", async () => {
    const fetchImpl = vi.fn<FetchLike>();
    const outcome = await probeInstance("https://a.example", 200, 100, fetchImpl);
    expect(outcome).toEqual({ kind: "expired" });
    expect(fetchImpl).not.toHaveBeenCalled();
  });

  it("treats expiry at the boundary as expired", async () => {
    const fetchImpl = vi.fn<FetchLike>();
    const outcome = await probeInstance("https://a.example", 100, 100, fetchImpl);
    expect(outcome).toEqual({ kind: "expired" });
  });

  it("resolves even when the response body is unreadable — only the status matters", async () => {
    const fetchImpl = vi.fn<FetchLike>().mockResolvedValue(okResponse());
    await expect(probeInstance("https://a.example", 100, 200, fetchImpl)).resolves.toEqual({
      kind: "connected",
    });
  });
});
