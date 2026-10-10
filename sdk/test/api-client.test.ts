import { describe, expect, it, mock } from "bun:test";
import { ApiClient, ApiError } from "../src/api-client.ts";

function fakeFetch(response: { status: number; body?: unknown }) {
  return mock(async (_input: string, _init?: RequestInit) => new Response(response.body !== undefined ? JSON.stringify(response.body) : null, { status: response.status }));
}

describe("ApiClient", () => {
  it("sends the token as a Bearer header and JSON-encodes the body", async () => {
    const fetchImpl = fakeFetch({ status: 200, body: { ok: true } });
    const api = new ApiClient({ baseUrl: "https://instance.example.com/", token: "dep_abc", fetchImpl });

    await api.request("POST", "/api/automations", { name: "x" });

    const [url, init] = fetchImpl.mock.calls[0]!;
    expect(url).toBe("https://instance.example.com/api/automations");
    expect((init!.headers as Record<string, string>).Authorization).toBe("Bearer dep_abc");
    expect(init!.body).toBe(JSON.stringify({ name: "x" }));
  });

  it("treats 204 as no content", async () => {
    const api = new ApiClient({ baseUrl: "https://x", token: "t", fetchImpl: fakeFetch({ status: 204 }) });
    await expect(api.request("DELETE", "/api/automations/1")).resolves.toBeUndefined();
  });

  it("throws ApiError with status and parsed body on a non-2xx response", async () => {
    const api = new ApiClient({ baseUrl: "https://x", token: "t", fetchImpl: fakeFetch({ status: 403, body: { error: "forbidden" } }) });
    await expect(api.request("GET", "/api/automations")).rejects.toMatchObject(
      new ApiError(403, { error: "forbidden" }, "GET /api/automations failed: 403"),
    );
  });

  it("carries the server's message in an ApiError, so run history says why", async () => {
    const body = { message: 'not found: no play named "Fix" in this workspace', code: "NOT_FOUND" };
    const api = new ApiClient({ baseUrl: "https://x", token: "t", fetchImpl: fakeFetch({ status: 404, body }) });
    await expect(api.request("POST", "/api/plays/queue", {})).rejects.toMatchObject(
      new ApiError(404, body, 'POST /api/plays/queue failed: 404: not found: no play named "Fix" in this workspace'),
    );
  });

  it("keeps a plain-text error body rather than failing to parse it", async () => {
    const fetchImpl = mock(async () => new Response("Bad Gateway", { status: 502 }));
    const api = new ApiClient({ baseUrl: "https://x", token: "t", fetchImpl });
    await expect(api.request("GET", "/api/automations")).rejects.toMatchObject(
      new ApiError(502, "Bad Gateway", "GET /api/automations failed: 502: Bad Gateway"),
    );
  });

  it("wires the automations version-push helper to the right path and payload", async () => {
    const fetchImpl = fakeFetch({ status: 201, body: { id: "v1" } });
    const api = new ApiClient({ baseUrl: "https://x", token: "t", fetchImpl });
    await api.automations.versions.push("auto-1", "console.log(1)", "initial");
    const [url, init] = fetchImpl.mock.calls[0]!;
    expect(url).toBe("https://x/api/automations/auto-1/versions");
    expect(JSON.parse(init!.body as string)).toEqual({ code: "console.log(1)", message: "initial" });
  });

  it("lists one workspace's automations when given one, and every readable workspace's without", async () => {
    const fetchImpl = fakeFetch({ status: 200, body: [] });
    const api = new ApiClient({ baseUrl: "https://x", token: "t", fetchImpl });
    await api.automations.list("ws-1");
    await api.automations.list();
    expect(fetchImpl.mock.calls.map(([url]) => url)).toEqual([
      "https://x/api/automations?workspace_id=ws-1",
      "https://x/api/automations",
    ]);
  });
});
