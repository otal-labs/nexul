// The only bridge to main: the renderer gets window.desktop (DesktopBridge), no Node, no ipcRenderer.
import { contextBridge, ipcRenderer } from "electron";

import type { ImportTokenResult, Result } from "./ipc";
import { IpcChannels } from "./ipc";
import type { PublicVaultState } from "./state";

contextBridge.exposeInMainWorld("desktop", {
  importToken: (token: string): Promise<ImportTokenResult> =>
    ipcRenderer.invoke(IpcChannels.importToken, token),
  listInstances: (): Promise<PublicVaultState> => ipcRenderer.invoke(IpcChannels.listInstances),
  connect: (id: string): Promise<Result> => ipcRenderer.invoke(IpcChannels.connect, id),
  remove: (id: string): Promise<Result> => ipcRenderer.invoke(IpcChannels.remove, id),
  refresh: (): Promise<void> => ipcRenderer.invoke(IpcChannels.refresh),
  onStateChanged: (callback: (state: PublicVaultState) => void): (() => void) => {
    const listener = (_event: Electron.IpcRendererEvent, state: PublicVaultState): void =>
      callback(state);
    ipcRenderer.on(IpcChannels.stateChanged, listener);
    return () => {
      ipcRenderer.removeListener(IpcChannels.stateChanged, listener);
    };
  },
});
