// Typed client over the automations-relevant HTTP surface. Every route
// here is gated by internal/auth's RequireAuth (session cookie or a `dep_`
// personal access token, see internal/auth/pats.go) — the automation's own
// `dat_` token only ever authenticates the WS dial-in endpoint
// (Service.AuthenticateToken in internal/automations/usecase.go has no path
// for it on the HTTP gateway).

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: unknown,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// FetchLike deliberately isn't `typeof fetch` — Bun's global fetch type
// carries Bun-specific extras (e.g. `preconnect`) that a plain mock or the
// standard DOM fetch signature doesn't have, and none of that is used here.
export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export interface ApiClientOptions {
  baseUrl: string;
  token: string;
  fetchImpl?: FetchLike;
}

export class ApiClient {
  private readonly baseUrl: string;
  private readonly token: string;
  private readonly fetchImpl: FetchLike;

  constructor(opts: ApiClientOptions) {
    this.baseUrl = opts.baseUrl.replace(/\/+$/, "");
    this.token = opts.token;
    this.fetchImpl = opts.fetchImpl ?? fetch;
  }

  async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const init: RequestInit = {
      method,
      headers: {
        Authorization: `Bearer ${this.token}`,
        ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      },
    };
    if (body !== undefined) init.body = JSON.stringify(body);
    const res = await this.fetchImpl(`${this.baseUrl}${path}`, init);
    if (res.status === 204) return undefined as T;
    const text = await res.text();
    const parsed = text ? JSON.parse(text) : undefined;
    if (!res.ok) {
      throw new ApiError(res.status, parsed, `${method} ${path} failed: ${res.status}`);
    }
    return parsed as T;
  }

  readonly automations = {
    list: <T = unknown>() => this.request<T>("GET", "/api/automations"),
    get: <T = unknown>(id: string) => this.request<T>("GET", `/api/automations/${id}`),
    updateConfig: <T = unknown>(id: string, configValues: unknown) =>
      this.request<T>("PATCH", `/api/automations/${id}/config`, { config_values: configValues }),
    setEnabled: <T = unknown>(id: string, enabled: boolean) =>
      this.request<T>("PATCH", `/api/automations/${id}/enabled`, { enabled }),
    versions: {
      push: <T = unknown>(id: string, code: string, message?: string) =>
        this.request<T>("POST", `/api/automations/${id}/versions`, { code, message }),
      list: <T = unknown>(id: string) => this.request<T>("GET", `/api/automations/${id}/versions`),
      get: <T = unknown>(id: string, versionId: string) => this.request<T>("GET", `/api/automations/${id}/versions/${versionId}`),
      diff: <T = unknown>(id: string) => this.request<T>("GET", `/api/automations/${id}/versions/diff`),
      merge: <T = unknown>(id: string, versionId: string) => this.request<T>("POST", `/api/automations/${id}/versions/${versionId}/merge`),
      rollback: <T = unknown>(id: string, versionId: string) =>
        this.request<T>("POST", `/api/automations/${id}/versions/${versionId}/rollback`),
    },
    runs: {
      list: <T = unknown>(id: string, limit?: number) =>
        this.request<T>("GET", `/api/automations/${id}/runs${limit ? `?limit=${limit}` : ""}`),
      get: <T = unknown>(id: string, runId: string) => this.request<T>("GET", `/api/automations/${id}/runs/${runId}`),
    },
  };

  readonly secrets = {
    list: <T = unknown>() => this.request<T>("GET", "/api/automation-secrets"),
    set: (name: string, value: string) => this.request<void>("PUT", `/api/automation-secrets/${name}`, { value }),
    delete: (name: string) => this.request<void>("DELETE", `/api/automation-secrets/${name}`),
  };
}
