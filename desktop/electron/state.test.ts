import { describe, expect, it } from "vitest";

import {
  activeInstance,
  connectionFor,
  initialState,
  publicState,
  vaultReducer,
  type InstanceEntry,
  type VaultState,
} from "./state";

const entry = (id: string, overrides: Partial<InstanceEntry> = {}): InstanceEntry => ({
  id,
  instanceUrl: `https://${id}.example`,
  token: `tok-${id}`,
  expiresAtSec: 1_800_000_000,
  addedAtSec: 1_700_000_000,
  ...overrides,
});

const add = (state: VaultState, e: InstanceEntry): VaultState =>
  vaultReducer(state, { type: "INSTANCE_ADDED", entry: e });

describe("vaultReducer — INSTANCE_ADDED", () => {
  it("adds an instance and auto-activates it on an empty vault", () => {
    const state = add(initialState(), entry("a"));
    expect(state.instances).toHaveLength(1);
    expect(state.activeId).toBe("a");
    expect(connectionFor(state, "a").status).toBe("idle");
  });

  it("does not steal the active slot when another instance is active", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = add(s1, entry("b"));
    expect(s2.instances).toHaveLength(2);
    expect(s2.activeId).toBe("a");
  });

  it("upserts on re-import, refreshing the token without duplicating", () => {
    const s1 = add(initialState(), entry("a", { token: "tok-old", expiresAtSec: 1 }));
    const s2 = add(s1, entry("a", { token: "tok-new", expiresAtSec: 2_000_000_000 }));
    expect(s2.instances).toHaveLength(1);
    expect(s2.instances[0]?.token).toBe("tok-new");
    expect(s2.instances[0]?.expiresAtSec).toBe(2_000_000_000);
    expect(s2.activeId).toBe("a");
  });
});

describe("vaultReducer — INSTANCE_REMOVED", () => {
  it("removes the instance and its connection state", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = vaultReducer(s1, { type: "CONNECT_FAILED", id: "a", error: "boom" });
    const s3 = vaultReducer(s2, { type: "INSTANCE_REMOVED", id: "a" });
    expect(s3.instances).toHaveLength(0);
    expect(s3.activeId).toBeNull();
    expect(s3.connections).toEqual({});
  });

  it("falls back to the first remaining instance when the active one is removed", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = add(s1, entry("b"));
    const s3 = vaultReducer(s2, { type: "ACTIVE_SET", id: "b" });
    const s4 = vaultReducer(s3, { type: "INSTANCE_REMOVED", id: "b" });
    expect(s4.activeId).toBe("a");
  });

  it("leaves the active id alone when a non-active instance is removed", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = add(s1, entry("b"));
    const s3 = vaultReducer(s2, { type: "INSTANCE_REMOVED", id: "b" });
    expect(s3.activeId).toBe("a");
  });
});

describe("vaultReducer — ACTIVE_SET", () => {
  it("sets the active instance", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = add(s1, entry("b"));
    const s3 = vaultReducer(s2, { type: "ACTIVE_SET", id: "b" });
    expect(s3.activeId).toBe("b");
  });

  it("is a no-op for an unknown id", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = vaultReducer(s1, { type: "ACTIVE_SET", id: "nope" });
    expect(s2).toBe(s1);
  });
});

describe("vaultReducer — connection transitions", () => {
  it("moves connecting → connected", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = vaultReducer(s1, { type: "CONNECT_STARTED", id: "a" });
    expect(connectionFor(s2, "a").status).toBe("connecting");
    const s3 = vaultReducer(s2, { type: "CONNECT_SUCCEEDED", id: "a" });
    expect(connectionFor(s3, "a").status).toBe("connected");
    expect(connectionFor(s3, "a").error).toBeNull();
  });

  it("moves connecting → unreachable carrying the error", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = vaultReducer(s1, { type: "CONNECT_STARTED", id: "a" });
    const s3 = vaultReducer(s2, { type: "CONNECT_FAILED", id: "a", error: "ECONNREFUSED" });
    expect(connectionFor(s3, "a").status).toBe("unreachable");
    expect(connectionFor(s3, "a").error).toBe("ECONNREFUSED");
  });

  it.each([
    ["CONNECT_STARTED", { type: "CONNECT_STARTED", id: "nope" } as const],
    ["CONNECT_SUCCEEDED", { type: "CONNECT_SUCCEEDED", id: "nope" } as const],
    ["CONNECT_FAILED", { type: "CONNECT_FAILED", id: "nope", error: "x" } as const],
  ])("ignores %s for an unknown id", (_name, action) => {
    const s1 = add(initialState(), entry("a"));
    const s2 = vaultReducer(s1, action);
    expect(s2).toBe(s1);
  });

  it("resets a stale connection to idle on re-import", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = vaultReducer(s1, { type: "CONNECT_FAILED", id: "a", error: "boom" });
    const s3 = add(s2, entry("a", { token: "fresh" }));
    expect(connectionFor(s3, "a").status).toBe("idle");
  });
});

describe("selectors", () => {
  it("activeInstance returns the active entry or null", () => {
    const s1 = add(initialState(), entry("a"));
    expect(activeInstance(s1)?.id).toBe("a");
    expect(activeInstance(initialState())).toBeNull();
  });

  it("connectionFor defaults to idle for an unknown instance", () => {
    expect(connectionFor(initialState(), "missing")).toEqual({ status: "idle", error: null });
  });

  it("publicState strips the token but keeps connection state and the active id", () => {
    const s1 = add(initialState(), entry("a"));
    const s2 = vaultReducer(s1, { type: "CONNECT_SUCCEEDED", id: "a" });
    const pub = publicState(s2);
    expect(pub.activeId).toBe("a");
    expect(pub.instances).toHaveLength(1);
    expect(pub.instances[0]).not.toHaveProperty("token");
    expect(pub.instances[0]?.connection.status).toBe("connected");
    expect(pub.instances[0]?.instanceUrl).toBe("https://a.example");
  });
});
