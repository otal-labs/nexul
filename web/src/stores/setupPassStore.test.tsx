import { beforeEach, describe, expect, it } from "vitest";

import { currentSetupPass, useSetupPassStore } from "@/stores/setupPassStore";

const pass = (expires_at: string) => ({ token: "pass-1", expires_at });

describe("setupPassStore", () => {
  beforeEach(() => {
    useSetupPassStore.setState({ token: null, expiresAt: null, code: null });
    localStorage.clear();
  });

  it("drops an expired pass on read so setup falls back to the code screen", () => {
    useSetupPassStore.getState().unlock(pass("2026-09-27T10:00:00Z"), "nxs_abc");
    expect(currentSetupPass(Date.parse("2026-09-27T10:00:01Z"))).toBeNull();
    expect(useSetupPassStore.getState().token).toBeNull();
  });

  it("returns a live pass", () => {
    useSetupPassStore.getState().unlock(pass("2026-09-27T10:00:00Z"), "nxs_abc");
    expect(currentSetupPass(Date.parse("2026-09-27T09:59:00Z"))).toBe("pass-1");
  });

  it("persists the pass under its own key but never the code", () => {
    useSetupPassStore.getState().unlock(pass("2026-09-27T10:00:00Z"), "nxs_abc");
    const stored = JSON.parse(localStorage.getItem("setup-pass") ?? "{}");
    expect(stored.state).toEqual({ token: "pass-1", expiresAt: "2026-09-27T10:00:00Z" });
    expect(useSetupPassStore.getState().code).toBe("nxs_abc");
  });
});
