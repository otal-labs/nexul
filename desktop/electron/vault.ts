// Connection states are transient (re-derived from probing) and never persisted, only instances/activeId.

import {
  initialState,
  type ConnectionInfo,
  type InstanceEntry,
  type VaultState,
} from "./state";

export interface KVStore {
  get(key: string): string | null;
  set(key: string, value: string): void;
}

const VAULT_KEY = "nexul.vault";

interface PersistedVault {
  instances?: unknown;
  activeId?: unknown;
}

export function loadVault(store: KVStore): VaultState {
  const raw = store.get(VAULT_KEY);
  if (raw === null) return initialState();
  let persisted: PersistedVault;
  try {
    persisted = JSON.parse(raw) as PersistedVault;
  } catch {
    return initialState();
  }
  const instances = Array.isArray(persisted.instances)
    ? persisted.instances.filter(isValidEntry)
    : [];
  const ids = new Set(instances.map((i) => i.id));
  const activeId =
    typeof persisted.activeId === "string" && ids.has(persisted.activeId)
      ? persisted.activeId
      : (instances[0]?.id ?? null);
  const connections: Record<string, ConnectionInfo> = {};
  for (const instance of instances) {
    connections[instance.id] = { status: "idle", error: null };
  }
  return { instances, activeId, connections };
}

export function saveVault(store: KVStore, state: VaultState): void {
  store.set(
    VAULT_KEY,
    JSON.stringify({ instances: state.instances, activeId: state.activeId }),
  );
}

function isValidEntry(value: unknown): value is InstanceEntry {
  if (typeof value !== "object" || value === null) return false;
  const entry = value as Record<string, unknown>;
  return (
    typeof entry.id === "string" &&
    typeof entry.instanceUrl === "string" &&
    typeof entry.token === "string" &&
    typeof entry.expiresAtSec === "number" &&
    typeof entry.addedAtSec === "number"
  );
}
