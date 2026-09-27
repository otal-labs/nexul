// AutomationTarget is one automation placed on this host, with the host-scoped token its worker dials in with.
export interface AutomationTarget {
  id: string;
  name: string;
  token: string;
}

export interface RemoteState {
  enabled: boolean;
  updatedAt: string;
  activeVersionId: string | null;
  activeCode: string | null;
}

// A removed host is refused with {"error":"automations_host_removed"}; it uninstalls itself rather than retry.
export type AssignmentsResult = { removed: true } | { removed: false; automations: AutomationTarget[] };

export interface AutomationsApi {
  fetchAssignments(serverUrl: string, credential: string): Promise<AssignmentsResult>;
  fetchState(target: AutomationTarget, serverUrl: string): Promise<RemoteState>;
}

interface AutomationResponse {
  enabled: boolean;
  updated_at: string;
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
    const res = await fetch(url, { headers: { Authorization: `Bearer ${credential}` } });
    if (res.status === 401 && (await res.text()).includes('"automations_host_removed"')) return { removed: true };
    if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`);
    const body = (await res.json()) as AssignmentsResponse;
    return { removed: false, automations: body.automations };
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
      updatedAt: automation.updated_at,
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
