// Renderer<->main IPC surface; result types stay plain serializable objects to cross the contextBridge.

import type { PublicVaultState } from "./state";

export const IpcChannels = {
  importToken: "desktop:import-token",
  listInstances: "desktop:list-instances",
  connect: "desktop:connect",
  remove: "desktop:remove",
  refresh: "desktop:refresh",
  stateChanged: "desktop:state-changed",
} as const;

export interface Result {
  readonly ok: boolean;
  readonly error?: string;
}

export interface ImportTokenResult extends Result {
  readonly id?: string;
}

// The only surface the renderer can reach; no direct Node/Electron access (contextIsolation + sandbox).
export interface DesktopBridge {
  importToken(token: string): Promise<ImportTokenResult>;
  listInstances(): Promise<PublicVaultState>;
  connect(id: string): Promise<Result>;
  remove(id: string): Promise<Result>;
  refresh(): Promise<void>;
  onStateChanged(callback: (state: PublicVaultState) => void): () => void;
}
