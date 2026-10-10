import type { SettingsStatusTone } from "@/components/settings/SettingsStatus";

// Mirrors pairing.Facts: what a computer's runner reports and what its T3 Code last listed. Only its owner reads them.
export interface ComputerFacts {
  hostname?: string;
  os?: string;
  arch?: string;
  runner_version?: string;
  t3: T3Facts;
  cloudflared?: string;
  git_name?: string;
  git_email?: string;
  free_disk_bytes?: number;
  providers: ProviderFacts[];
  projects: { id: string; title: string; path: string }[];
}

export type T3State = "answering" | "not_running" | "missing" | "not_loopback";

export interface T3Facts {
  state: T3State;
  install?: "service" | "command_line" | "desktop_app";
  port?: number;
  version?: string;
  // When the runner last restarted T3 Code's background service, and why it could not.
  restarted_at?: string;
  restart_error?: string;
}

export interface ProviderFacts {
  id: string;
  driver: string;
  name: string;
  version?: string;
  sign_in: "signed_in" | "signed_out" | "unknown";
  models: { slug: string; name: string }[];
}

// The relay address a computer with a personal runner is reached at; people never need to see it.
export const RUNNER_HOST_SUFFIX = ".nexul-computer.invalid";

export const reachedThroughRunner = (serverURL: string) => serverURL.endsWith(RUNNER_HOST_SUFFIX);

const T3_STATE: Record<T3State, { tone: SettingsStatusTone; text: string }> = {
  answering: { tone: "success", text: "Running" },
  not_running: { tone: "warning", text: "Not running" },
  missing: { tone: "destructive", text: "Not installed" },
  not_loopback: { tone: "destructive", text: "Listens beyond this computer" },
};

const INSTALL_LABEL: Record<NonNullable<T3Facts["install"]>, string> = {
  service: "background service",
  command_line: "command line",
  desktop_app: "desktop app",
};

// T3 Code's state as the row shows it: a closed desktop app asks to be opened, and a running one names its port.
export const t3Status = (t3: T3Facts): { tone: SettingsStatusTone; text: string; detail: string } => {
  const state = T3_STATE[t3.state];
  const opened = t3.state === "not_running" && t3.install === "desktop_app" ? "Open T3 Code" : "";
  const detail = [opened, t3.port && `port ${t3.port}`, t3.version, t3.install && INSTALL_LABEL[t3.install]].filter(Boolean).join(" · ");
  return { ...state, detail };
};

export const SIGN_IN: Record<ProviderFacts["sign_in"], { tone: SettingsStatusTone; text: string }> = {
  signed_in: { tone: "success", text: "Signed in" },
  signed_out: { tone: "warning", text: "Signed out" },
  unknown: { tone: "muted", text: "Sign-in unknown" },
};
