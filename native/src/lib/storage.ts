import Storage from "expo-sqlite/kv-store";
import type { StateStorage } from "zustand/middleware";

// Synchronous reads hydrate a persisted store before first render, so no flash of defaults.
export const kvStateStorage: StateStorage = {
  getItem: (name) => Storage.getItemSync(name),
  setItem: (name, value) => Storage.setItemSync(name, value),
  removeItem: (name) => Storage.removeItemSync(name),
};
