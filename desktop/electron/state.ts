// A pure reducer: the main process owns IO, this owns transitions; status always comes from probe.ts.

export type ConnectionStatus = "idle" | "connecting" | "connected" | "unreachable" | "expired";

export interface ConnectionInfo {
  readonly status: ConnectionStatus;
  readonly error: string | null;
}

export interface InstanceEntry {
  readonly id: string;
  readonly instanceUrl: string;
  readonly token: string;
  readonly expiresAtSec: number;
  readonly addedAtSec: number;
}

export interface VaultState {
  readonly instances: readonly InstanceEntry[];
  readonly activeId: string | null;
  readonly connections: Readonly<Record<string, ConnectionInfo>>;
}

export type VaultAction =
  | { readonly type: "INSTANCE_ADDED"; readonly entry: InstanceEntry }
  | { readonly type: "INSTANCE_REMOVED"; readonly id: string }
  | { readonly type: "ACTIVE_SET"; readonly id: string }
  | { readonly type: "CONNECT_STARTED"; readonly id: string }
  | { readonly type: "CONNECT_SUCCEEDED"; readonly id: string }
  | { readonly type: "CONNECT_FAILED"; readonly id: string; readonly error: string };

export function initialState(): VaultState {
  return { instances: [], activeId: null, connections: {} };
}

export function vaultReducer(state: VaultState, action: VaultAction): VaultState {
  switch (action.type) {
    case "INSTANCE_ADDED": {
      // Upsert: re-importing the same origin keeps the stable id but refreshes the token/expiry.
      const exists = state.instances.some((i) => i.id === action.entry.id);
      const instances = exists
        ? state.instances.map((i) => (i.id === action.entry.id ? action.entry : i))
        : [...state.instances, action.entry];
      let activeId = state.activeId;
      if (!exists && activeId === null) activeId = action.entry.id;
      return {
        instances,
        activeId,
        connections: {
          ...state.connections,
          [action.entry.id]: { status: "idle", error: null },
        },
      };
    }
    case "INSTANCE_REMOVED": {
      const instances = state.instances.filter((i) => i.id !== action.id);
      const connections = { ...state.connections };
      delete connections[action.id];
      let activeId = state.activeId;
      if (activeId === action.id) activeId = instances[0]?.id ?? null;
      return { instances, activeId, connections };
    }
    case "ACTIVE_SET":
      if (!state.instances.some((i) => i.id === action.id)) return state;
      return { ...state, activeId: action.id };
    case "CONNECT_STARTED":
      return withConnection(state, action.id, { status: "connecting", error: null });
    case "CONNECT_SUCCEEDED":
      return withConnection(state, action.id, { status: "connected", error: null });
    case "CONNECT_FAILED":
      return withConnection(state, action.id, { status: "unreachable", error: action.error });
  }
}

export function activeInstance(state: VaultState): InstanceEntry | null {
  return state.instances.find((i) => i.id === state.activeId) ?? null;
}

export function connectionFor(state: VaultState, id: string): ConnectionInfo {
  return state.connections[id] ?? { status: "idle", error: null };
}

// The token is never shipped across the bridge to the renderer; it has no use for it.
export interface PublicInstance {
  readonly id: string;
  readonly instanceUrl: string;
  readonly expiresAtSec: number;
  readonly addedAtSec: number;
  readonly connection: ConnectionInfo;
}

export interface PublicVaultState {
  readonly instances: readonly PublicInstance[];
  readonly activeId: string | null;
}

export function publicState(state: VaultState): PublicVaultState {
  return {
    activeId: state.activeId,
    instances: state.instances.map((entry) => ({
      id: entry.id,
      instanceUrl: entry.instanceUrl,
      expiresAtSec: entry.expiresAtSec,
      addedAtSec: entry.addedAtSec,
      connection: connectionFor(state, entry.id),
    })),
  };
}

function withConnection(
  state: VaultState,
  id: string,
  info: ConnectionInfo,
): VaultState {
  if (!state.instances.some((i) => i.id === id)) return state;
  return { ...state, connections: { ...state.connections, [id]: info } };
}
