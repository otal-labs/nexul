// Composition root (ws-30): domain logic lives in token/state/vault/probe.ts; this just wires them to Electron.
import { app, BrowserWindow, dialog, ipcMain, Menu, session } from "electron";
import type { MenuItemConstructorOptions } from "electron";
import fs from "node:fs";
import path from "node:path";

import type { ImportTokenResult, Result } from "./ipc";
import { IpcChannels } from "./ipc";
import { instanceIdFor, parseConnectionToken } from "./token";
import { probeInstance } from "./probe";
import {
  activeInstance,
  initialState,
  publicState,
  vaultReducer,
  type InstanceEntry,
  type PublicVaultState,
  type VaultState,
} from "./state";
import { loadVault, saveVault, type KVStore } from "./vault";
import { isAllowedNavigation } from "./navigation";

// The shell's own page; everything else the window shows is the server's SPA at the active instance origin.
const LAUNCHER_PATH = path.join(__dirname, "..", "dist", "index.html");

// Named partition so the instance session (auth token, cookies) survives restarts, isolated per-app.
const INSTANCE_PARTITION = "persist:nexul";

const nowSec = (): number => Math.floor(Date.now() / 1000);

let win: BrowserWindow | null = null;
let vault: VaultState = initialState();
let kv: KVStore;

app.whenReady().then(() => {
  kv = fileKV(path.join(app.getPath("userData"), "vault.json"));
  vault = loadVault(kv);
  registerIpc();
  buildMenu();
  createWindow();
  const entry = activeInstance(vault);
  if (entry !== null) void loadInstance(entry);
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

app.on("activate", () => {
  if (BrowserWindow.getAllWindows().length === 0) createWindow();
});

function createWindow(): void {
  win = new BrowserWindow({
    width: 1200,
    height: 800,
    show: false,
    title: "Nexul",
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      session: session.fromPartition(INSTANCE_PARTITION),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
      allowRunningInsecureContent: false,
    },
  });
  win.once("ready-to-show", () => win?.show());
  // Popups are never needed because the OAuth dance happens in the same frame, so window.open is denied outright.
  win.webContents.setWindowOpenHandler(() => ({ action: "deny" }));
  win.webContents.on("will-navigate", (event, url) => {
    const current = win?.webContents.getURL() ?? "";
    const activeOrigin = activeInstance(vault)?.instanceUrl ?? null;
    // Only the launcher, the active instance, or github.com (AU1's OAuth dance) may load here.
    if (!isAllowedNavigation(url, current, activeOrigin)) {
      event.preventDefault();
    }
  });
  win.on("closed", () => {
    win = null;
  });
  void win.loadFile(LAUNCHER_PATH);
}

function showLauncher(): void {
  if (win === null) return;
  void win.loadFile(LAUNCHER_PATH);
}

function registerIpc(): void {
  ipcMain.handle(IpcChannels.importToken, (_event, rawToken: unknown): ImportTokenResult => {
    if (typeof rawToken !== "string" || rawToken.trim() === "") {
      return { ok: false, error: "no token given" };
    }
    const parsed = parseConnectionToken(rawToken.trim());
    if (!parsed.ok) return { ok: false, error: parsed.error };
    const entry: InstanceEntry = {
      id: instanceIdFor(parsed.claims.instanceUrl),
      instanceUrl: parsed.claims.instanceUrl,
      token: rawToken.trim(),
      expiresAtSec: parsed.claims.expiresAtSec,
      addedAtSec: nowSec(),
    };
    dispatch({ type: "INSTANCE_ADDED", entry });
    saveVault(kv, vault);
    broadcast();
    return { ok: true, id: entry.id };
  });

  ipcMain.handle(IpcChannels.listInstances, (): PublicVaultState => publicState(vault));

  ipcMain.handle(IpcChannels.connect, (_event, id: unknown): Result => {
    if (typeof id !== "string") return { ok: false, error: "bad instance id" };
    const entry = vault.instances.find((i) => i.id === id);
    if (entry === undefined) return { ok: false, error: "unknown instance" };
    void loadInstance(entry);
    return { ok: true };
  });

  ipcMain.handle(IpcChannels.remove, (_event, id: unknown): Result => {
    if (typeof id !== "string") return { ok: false, error: "bad instance id" };
    const wasActive = vault.activeId === id;
    dispatch({ type: "INSTANCE_REMOVED", id });
    saveVault(kv, vault);
    broadcast();
    if (wasActive) showLauncher();
    return { ok: true };
  });

  ipcMain.handle(IpcChannels.refresh, (): Promise<void> => refreshAll());
}

// Navigates the window only once the probe succeeds; on failure the launcher stays "unreachable" to retry.
async function loadInstance(entry: InstanceEntry): Promise<void> {
  dispatch({ type: "ACTIVE_SET", id: entry.id });
  dispatch({ type: "CONNECT_STARTED", id: entry.id });
  saveVault(kv, vault);
  broadcast();
  const outcome = await probeInstance(entry.instanceUrl, nowSec(), entry.expiresAtSec);
  if (outcome.kind === "connected") {
    dispatch({ type: "CONNECT_SUCCEEDED", id: entry.id });
    broadcast();
    if (win !== null) void win.loadURL(entry.instanceUrl);
    return;
  }
  dispatch({
    type: "CONNECT_FAILED",
    id: entry.id,
    error:
      outcome.kind === "expired"
        ? "connection token has expired — re-import it"
        : outcome.error,
  });
  broadcast();
  showLauncher();
}

async function refreshAll(): Promise<void> {
  const ids = vault.instances.map((i) => i.id);
  for (const id of ids) dispatch({ type: "CONNECT_STARTED", id });
  broadcast();
  await Promise.all(
    ids.map(async (id) => {
      const entry = vault.instances.find((i) => i.id === id);
      if (entry === undefined) return;
      const outcome = await probeInstance(entry.instanceUrl, nowSec(), entry.expiresAtSec);
      dispatch(
        outcome.kind === "connected"
          ? { type: "CONNECT_SUCCEEDED", id }
          : {
              type: "CONNECT_FAILED",
              id,
              error:
                outcome.kind === "expired"
                  ? "connection token has expired — re-import it"
                  : outcome.error,
            },
      );
    }),
  );
  broadcast();
}

function buildMenu(): void {
  const template: MenuItemConstructorOptions[] = [
    ...(process.platform === "darwin" ? [{ role: "appMenu" as const }] : []),
    {
      label: "Instances",
      submenu: [
        { label: "Add instance…", click: () => showLauncher() },
        { type: "separator" },
        ...(vault.instances.length === 0
          ? [{ label: "No instances yet", enabled: false }]
          : vault.instances.map((entry) => ({
              label: entry.instanceUrl,
              type: "radio" as const,
              checked: entry.id === vault.activeId,
              click: () => void loadInstance(entry),
            }))),
        { type: "separator" },
        { label: "Manage instances", click: () => showLauncher() },
        {
          label: "Remove current instance…",
          enabled: vault.activeId !== null,
          click: () => void removeActive(),
        },
      ],
    },
    {
      label: "View",
      submenu: [
        { role: "reload" },
        { role: "forceReload" },
        { role: "toggleDevTools" },
        { type: "separator" },
        { role: "resetZoom" },
        { role: "zoomIn" },
        { role: "zoomOut" },
      ],
    },
  ];
  Menu.setApplicationMenu(Menu.buildFromTemplate(template));
}

async function removeActive(): Promise<void> {
  const entry = activeInstance(vault);
  if (entry === null || win === null) return;
  const { response } = await dialog.showMessageBox(win, {
    type: "question",
    buttons: ["Remove", "Cancel"],
    defaultId: 1,
    cancelId: 1,
    message: `Remove ${entry.instanceUrl} from this app?`,
    detail: "Its connection token is deleted from the vault. You can re-import it later.",
  });
  if (response !== 0) return;
  dispatch({ type: "INSTANCE_REMOVED", id: entry.id });
  saveVault(kv, vault);
  broadcast();
  showLauncher();
}

function dispatch(action: Parameters<typeof vaultReducer>[1]): void {
  vault = vaultReducer(vault, action);
}

function broadcast(): void {
  win?.webContents.send(IpcChannels.stateChanged, publicState(vault));
  buildMenu();
}

// Adapts one JSON file to vault.ts's KVStore; the vault stores a single key, so the file is its value.
function fileKV(filePath: string): KVStore {
  return {
    get: () => {
      try {
        return fs.readFileSync(filePath, "utf8");
      } catch {
        return null;
      }
    },
    set: (_key: string, value: string) => fs.writeFileSync(filePath, value, "utf8"),
  };
}
