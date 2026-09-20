import { describe, expect, it } from "vitest";

import {
  BASE_URL,
  E2E_PROJECT_ID,
  E2E_WORKSPACE_ID,
  MCP_URL,
  authedHeaders,
  makeApiClient,
  mintToken,
} from "../helpers";

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

const json = <T = unknown>(body: T) => ({ body: JSON.stringify(body) });

describe("api (HTTP gateway)", () => {
  it("rejects unauthenticated requests", async () => {
    const res = await fetch(`${BASE_URL}/api/docs`);
    expect(res.status).toBe(401);
  });

  it("serves the current user", async () => {
    const { status, body } = await api<{ user: { login: string } }>("/api/auth/me");
    expect(status).toBe(200);
    expect(body?.user.login).toBe("e2e-user");
  });

  it("docs: create → get → update → search → delete", async () => {
    const created = await api<{ id: string }>("/api/docs", {
      method: "POST",
      ...json({ title: `e2e-api-doc-${Date.now()}`, project_id: E2E_PROJECT_ID }),
    });
    expect(created.status).toBe(201);
    const id = created.body!.id;

    expect((await api(`/api/docs/${id}`)).status).toBe(200);
    expect((await api(`/api/docs/${id}`, { method: "PUT", ...json({ title: "updated" }) })).status).toBe(200);
    expect((await api("/api/docs/search?q=e2e-api-doc")).status).toBe(200);
    expect((await api(`/api/docs/${id}`, { method: "DELETE" })).status).toBe(204);
  });

  it("tickets: create → label → status → delete", async () => {
    const created = await api<{ id: string }>("/api/tickets", {
      method: "POST",
      ...json({ title: `e2e-api-ticket-${Date.now()}`, project_id: E2E_PROJECT_ID }),
    });
    expect(created.status).toBe(201);
    const id = created.body!.id;

    expect((await api(`/api/tickets/${id}/labels`, { method: "POST", ...json({ label: "e2e" }) })).status).toBe(200);
    expect((await api(`/api/tickets/${id}/status`, { method: "PATCH", ...json({ status: "in_progress" }) })).status).toBe(200);
    expect((await api(`/api/tickets/${id}`, { method: "DELETE" })).status).toBe(204);
  });

  it("board config: projects / categories / statuses / ticket-types", async () => {
    expect((await api(`/api/projects?workspace_id=${E2E_WORKSPACE_ID}`)).status).toBe(200);
    const cat = await api<{ id: string }>("/api/categories", {
      method: "POST",
      ...json({ project_id: E2E_PROJECT_ID, name: `e2e-cat-${Date.now()}` }),
    });
    expect(cat.status).toBe(201);
    expect((await api(`/api/categories/${cat.body!.id}`, { method: "DELETE" })).status).toBe(204);
    expect((await api("/api/statuses")).status).toBe(200);
    expect((await api("/api/ticket-types")).status).toBe(200);
  });

  it("services: create → update → deploy enqueue → cancel → delete", async () => {
    const svc = await api<{ id: string }>("/api/services", {
      method: "POST",
      ...json({
        project_id: E2E_PROJECT_ID,
        name: `e2e-api-svc-${Date.now()}`,
        target: "e2e-host",
        strategy: "run",
        docker_network: "e2e-net",
        health_check: { url: "http://e2e/health" },
      }),
    });
    expect(svc.status).toBe(201);
    const id = svc.body!.id;

    const dep = await api<{ id: string }>("/api/deploys", {
      method: "POST",
      ...json({ service_id: id, image: "alpine" }),
    });
    expect(dep.status).toBe(201);
    expect((await api(`/api/deploys/${dep.body!.id}/cancel`, { method: "POST" })).status).toBe(202);
    expect((await api(`/api/services/${id}`, { method: "DELETE" })).status).toBe(200);
  });

  it("automations: create → config → enable → mint/revoke token → delete", async () => {
    expect((await api("/api/automations")).status).toBe(200);
    const created = await api<{ automation: { id: string }; token: string }>("/api/automations", {
      method: "POST",
      ...json({ name: `e2e-automation-${Date.now()}`, scopes: ["tickets:read"] }),
    });
    expect(created.status).toBe(201);
    expect(created.body!.token).toBeTruthy();
    const id = created.body!.automation.id;

    expect((await api(`/api/automations/${id}`)).status).toBe(200);
    expect(
      (await api(`/api/automations/${id}/config`, { method: "PATCH", ...json({ config_values: { status: "done" } }) }))
        .status,
    ).toBe(200);
    expect((await api(`/api/automations/${id}/enabled`, { method: "PATCH", ...json({ enabled: true }) })).status).toBe(
      200,
    );
    expect((await api(`/api/automations/${id}/token`, { method: "POST" })).status).toBe(201);
    expect((await api(`/api/automations/${id}/token`, { method: "DELETE" })).status).toBe(200);
    expect((await api(`/api/automations/${id}`, { method: "DELETE" })).status).toBe(204);
  });

  it("workspace surface: notifications, runners, permissions, mentions, audit, events, integrations", async () => {
    expect((await api("/api/notifications")).status).toBe(200);
    expect((await api("/api/notifications/unread-count")).status).toBe(200);
    expect((await api("/api/runners")).status).toBe(200);
    expect((await api("/api/permissions/users")).status).toBe(200);
    expect((await api("/api/mentions/search?q=e2e")).status).toBe(200);
    expect((await api("/api/audit")).status).toBe(200);
    expect((await api("/api/events/catalog")).status).toBe(200);
    expect((await api("/api/integrations")).status).toBe(200);
    expect((await api("/api/dns/status")).status).toBe(200);
  });

  it("topology: get + round-trip put", async () => {
    const got = await api<Record<string, unknown>>("/api/topology");
    expect(got.status).toBe(200);
    expect((await api("/api/topology", { method: "PUT", ...json(got.body) })).status).toBe(200);
  });
});

describe("mcp", () => {
  it("handshakes and lists tools as the authenticated user", async () => {
    const init = await fetch(MCP_URL, {
      method: "POST",
      headers: authedHeaders(TOKEN),
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "e2e", version: "1" } },
      }),
    });
    expect(init.status).toBe(200);
    const initBody = (await init.json()) as { result?: { serverInfo?: { name?: string } } };
    expect(initBody.result?.serverInfo?.name).toBeDefined();

    const tools = await fetch(MCP_URL, {
      method: "POST",
      headers: authedHeaders(TOKEN),
      body: JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} }),
    });
    expect(tools.status).toBe(200);
    const toolsBody = (await tools.json()) as { result?: { tools?: { name: string }[] } };
    const names = toolsBody.result?.tools?.map((t) => t.name) ?? [];
    for (const expected of ["doc_search", "ticket_search", "topology_get"]) {
      expect(names).toContain(expected);
    }
  });

  it("rejects unauthenticated MCP requests", async () => {
    const res = await fetch(MCP_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ jsonrpc: "2.0", id: 3, method: "tools/list", params: {} }),
    });
    expect(res.status).toBe(401);
  });
});
