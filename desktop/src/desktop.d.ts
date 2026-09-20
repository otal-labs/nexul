/// <reference types="vite/client" />

import type { PublicVaultState } from "../electron/state";
import type { DesktopBridge } from "../electron/ipc";

declare global {
  interface Window {
    desktop: DesktopBridge;
  }
}

export type { PublicVaultState };
