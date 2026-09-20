import type { AutomationTarget } from "./host-tokens.ts";

export interface RemoteState {
  enabled: boolean;
  updatedAt: string;
  activeVersionId: string | null;
  activeCode: string | null;
}

export interface AutomationsApi {
  fetchState(target: AutomationTarget, serverUrl: string): Promise<RemoteState>;
}

interface AutomationResponse {
  enabled: boolean;
  updated_at: string;
}

interface VersionDiffResponse {
  active: { id: string; code: string } | null;
}

// httpAutomationsApi hits the real HTTP gateway with the automation's own
// dat_ token (ADR 0046). Known limitation (ticket 11's comments, resolved by
// ticket 16 concurrently): dat_ tokens don't authenticate on the HTTP
// gateway yet, only the WS dial-in endpoint — this call is written as if
// that gap is closed, matching the two default automations' ctx.api calls.
export const httpAutomationsApi: AutomationsApi = {
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
