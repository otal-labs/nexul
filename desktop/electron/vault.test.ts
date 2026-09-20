import { describe, expect, it } from "vitest";

import { initialState, vaultReducer } from "./state";
import { loadVault, saveVault, type KVStore } from "./vault";

class MemoryKV implements KVStore {
  private readonly data = new Map<string, string>();
  get(key: string): string | null {
    return this.data.get(key) ?? null;
  }
  set(key: string, value: string): void {
    this.data.set(key, value);
  }
  raw(): Map<string, string> {
    return this.data;
  }
}

const persisted = (raw: string): KVStore => {
  const store = new MemoryKV();
  store.set("nexul.vault", raw);
  return store;
};

describe("loadVault", () => {
  it("returns an empty vault when nothing is stored", () => {
    const state = loadVault(new MemoryKV());
    expect(state.instances).toEqual([]);
    expect(state.activeId).toBeNull();
    expect(state.connections).toEqual({});
  });

  it("returns an empty vault on corrupted JSON", () => {
    const state = loadVault(persisted("{not json"));
    expect(state.instances).toEqual([]);
    expect(state.activeId).toBeNull();
  });

  it("returns an empty vault when instances is not an array", () => {
    const state = loadVault(persisted(JSON.stringify({ instances: "nope", activeId: null })));
    expect(state.instances).toEqual([]);
  });

  it("restores instances and the active id, with idle connections", () => {
    const raw = JSON.stringify({
      instances: [
        {
          id: "a",
          instanceUrl: "https://a.example",
          token: "tok-a",
          expiresAtSec: 1_800_000_000,
          addedAtSec: 1_700_000_000,
        },
        {
          id: "b",
          instanceUrl: "https://b.example",
          token: "tok-b",
          expiresAtSec: 1_800_000_000,
          addedAtSec: 1_700_000_000,
        },
      ],
      activeId: "b",
    });
    const state = loadVault(persisted(raw));
    expect(state.instances).toHaveLength(2);
    expect(state.activeId).toBe("b");
    for (const instance of state.instances) {
      expect(state.connections[instance.id]).toEqual({ status: "idle", error: null });
    }
  });

  it("drops entries missing required fields", () => {
    const raw = JSON.stringify({
      instances: [
        { id: "a", instanceUrl: "https://a.example", token: "t", expiresAtSec: 1, addedAtSec: 1 },
        { id: "b", instanceUrl: "https://b.example", expiresAtSec: 1, addedAtSec: 1 },
        { id: "c", instanceUrl: 42, token: "t", expiresAtSec: 1, addedAtSec: 1 },
        null,
        "junk",
      ],
      activeId: "b",
    });
    const state = loadVault(persisted(raw));
    expect(state.instances.map((i) => i.id)).toEqual(["a"]);
    expect(state.activeId).toBe("a");
  });

  it("repairs an active id pointing at a missing instance", () => {
    const raw = JSON.stringify({
      instances: [
        { id: "a", instanceUrl: "https://a.example", token: "t", expiresAtSec: 1, addedAtSec: 1 },
      ],
      activeId: "ghost",
    });
    const state = loadVault(persisted(raw));
    expect(state.activeId).toBe("a");
  });
});

describe("saveVault / roundtrip", () => {
  it("persists instances and the active id", () => {
    const store = new MemoryKV();
    let state = initialState();
    state = vaultReducer(state, {
      type: "INSTANCE_ADDED",
      entry: {
        id: "a",
        instanceUrl: "https://a.example",
        token: "tok-a",
        expiresAtSec: 1_800_000_000,
        addedAtSec: 1_700_000_000,
      },
    });
    state = vaultReducer(state, { type: "CONNECT_FAILED", id: "a", error: "boom" });
    saveVault(store, state);

    const reloaded = loadVault(store);
    expect(reloaded.activeId).toBe("a");
    expect(reloaded.instances[0]?.token).toBe("tok-a");
    expect(reloaded.instances[0]?.instanceUrl).toBe("https://a.example");
  });

  it("never persists connection state — it is re-derived by probing", () => {
    const store = new MemoryKV();
    let state = initialState();
    state = vaultReducer(state, {
      type: "INSTANCE_ADDED",
      entry: {
        id: "a",
        instanceUrl: "https://a.example",
        token: "t",
        expiresAtSec: 1_800_000_000,
        addedAtSec: 1_700_000_000,
      },
    });
    state = vaultReducer(state, { type: "CONNECT_SUCCEEDED", id: "a" });
    saveVault(store, state);
    const serialized = store.raw().get("nexul.vault");
    expect(serialized).toBeDefined();
    expect(JSON.parse(serialized ?? "{}")).not.toHaveProperty("connections");
  });

  it("survives a full save → load → save cycle without losing entries", () => {
    const store = new MemoryKV();
    let state = initialState();
    for (const id of ["a", "b", "c"]) {
      state = vaultReducer(state, {
        type: "INSTANCE_ADDED",
        entry: {
          id,
          instanceUrl: `https://${id}.example`,
          token: `tok-${id}`,
          expiresAtSec: 1_800_000_000,
          addedAtSec: 1_700_000_000,
        },
      });
    }
    saveVault(store, state);
    const reloaded = loadVault(store);
    saveVault(store, reloaded);
    const again = loadVault(store);
    expect(again.instances.map((i) => i.id).sort()).toEqual(["a", "b", "c"]);
  });
});
