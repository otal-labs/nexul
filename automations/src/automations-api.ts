import type { ManagerState } from "./resolver.ts";
import { hostVersion } from "./version.ts";

// AutomationTarget is one automation placed on this host, with the host-scoped token its worker dials in with and its
// workspace: the id names the secrets manager token the host resolves its pool with, the slug is for logs.
export interface AutomationTarget {
  id: string;
  name: string;
  token: string;
  workspace_id: string;
  workspace: string;
}

// A secrets manager this host holds a token for in one workspace, by its id; manager is named by its reference scheme ("op").
export interface ManagerReport {
  workspace_id: string;
  manager: string;
  state: ManagerState;
}

export interface RemoteState {
  enabled: boolean;
  // The config values as JSON text, compared to tell when the worker's hello is out of date.
  configValues: string;
  activeVersionId: string | null;
  activeCode: string | null;
}

// A removed host is refused with {"error":"automations_host_removed"}; it uninstalls itself rather than retry.
export type AssignmentsResult = { removed: true } | { removed: false; automations: AutomationTarget[] };

export interface AutomationsApi {
  fetchAssignments(serverUrl: string, credential: string): Promise<AssignmentsResult>;
  fetchState(target: AutomationTarget, serverUrl: string): Promise<RemoteState>;
  // Replaces what the server holds about this host's secrets managers.
  reportManagers(serverUrl: string, credential: string, managers: ManagerReport[]): Promise<void>;
}

interface AutomationResponse {
  enabled: boolean;
  config_values: unknown;
}

interface VersionDiffResponse {
  active: { id: string; code: string } | null;
}

interface AssignmentsResponse {
  automations: AutomationTarget[];
}

export const httpAutomationsApi: AutomationsApi = {
  async fetchAssignments(serverUrl, credential) {
    const url = `${serverUrl}/api/automation-hosts/self/assignments`;
    const res = await fetch(`${url}?version=${encodeURIComponent(hostVersion)}`, { headers: { Authorization: `Bearer ${credential}` } });
    if (res.status === 401 && (await res.text()).includes('"automations_host_removed"')) return { removed: true };
    if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`);
    const body = (await res.json()) as AssignmentsResponse;
    return { removed: false, automations: body.automations };
  },

  async reportManagers(serverUrl, credential, managers) {
    const url = `${serverUrl}/api/automation-hosts/self/managers`;
    const res = await fetch(url, {
      method: "POST",
      headers: { Authorization: `Bearer ${credential}`, "Content-Type": "application/json" },
      body: JSON.stringify({ managers }),
    });
    if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`);
  },

  // The worker's own token reads its automation: a self-read needs no scope (CONTEXT.md, Self-read).
  async fetchState(target, serverUrl) {
    const headers = { Authorization: `Bearer ${target.token}` };
    const [automation, diff] = await Promise.all([
      fetchJSON<AutomationResponse>(`${serverUrl}/api/automations/${target.id}`, headers),
      fetchJSON<VersionDiffResponse>(`${serverUrl}/api/automations/${target.id}/versions/diff`, headers),
    ]);
    return {
      enabled: automation.enabled,
      configValues: JSON.stringify(automation.config_values ?? null),
      activeVersionId: diff.active?.id ?? null,
      activeCode: diff.active?.code ?? null,
    };
  },
};

async function fetchJSON<T>(url: string, headers: Record<string, string>): Promise<T> {
  const res = await fetch(url, { headers });
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`);
  return (await res.json()) as T;
}
