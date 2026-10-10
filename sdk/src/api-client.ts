// Typed client over the HTTP gateway, as a `dep_` personal access token (the CLI) or an automation's `dat_` token.

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
    if (!res.ok) {
      const body = parseBody(text);
      throw new ApiError(res.status, body, `${method} ${path} failed: ${res.status}${serverText(body, text)}`);
    }
    return (text ? JSON.parse(text) : undefined) as T;
  }

  readonly automations = {
    // Without a workspace it lists every workspace's automations the token's creator can read.
    list: <T = unknown>(workspaceId?: string) =>
      this.request<T>("GET", `/api/automations${workspaceId ? `?workspace_id=${encodeURIComponent(workspaceId)}` : ""}`),
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

// parseBody reads an error body as JSON, falling back to its text, since a proxy in front may answer in plain text.
function parseBody(text: string): unknown {
  if (!text) return undefined;
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

// serverText is the gateway's own message, so run history says why ("no play named …"), not only the status.
function serverText(body: unknown, text: string): string {
  const message = typeof body === "object" && body !== null && "message" in body ? body.message : undefined;
  if (typeof message === "string" && message) return `: ${message}`;
  return typeof body === "string" && text ? `: ${text.slice(0, 200)}` : "";
}
