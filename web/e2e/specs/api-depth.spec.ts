import { describe, expect, it } from "vitest";

import { E2E_PROJECT_ID, makeApiClient, mintToken } from "../helpers";

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

const json = <T = unknown>(body: T) => ({ body: JSON.stringify(body) });

describe("api depth (HTTP gateway)", () => {
  it("service hostnames: read surface works, write path fails closed without a provider", async () => {
    const svc = await api<{ id: string }>("/api/services", {
      method: "POST",
      ...json({
        project_id: E2E_PROJECT_ID,
        name: `e2e-host-svc-${Date.now()}`,
        target: "e2e-host",
        strategy: "run",
        docker_network: "e2e-net",
        health_check: { url: "http://e2e/health" },
      }),
    });
    expect(svc.status).toBe(201);
    const serviceId = svc.body!.id;
    const hostname = `api-${Date.now()}.e2e.test`;

    try {
      // The read surface always answers: the list is empty, and a service without a hostname answers 404 (not-found, never an auth failure).
      const list = await api<{ hostname: string; service: string }[]>("/api/dns/service-hostnames");
      expect(list.status).toBe(200);
      expect(
        (await api<{ hostname: string }>(`/api/dns/service-hostnames/${serviceId}`)).status,
      ).toBe(404);

      // Writing requires a connected DNS provider: without one the gateway answers 503, never 401, since a provider gate must not sign the session out.
      const set = await api("/api/dns/service-hostnames", {
        method: "POST",
        ...json({
          service: serviceId,
          hostname,
          zone_id: "z-e2e",
          zone: "e2e.test",
          type: "CNAME",
          target: "svc.e2e.test",
        }),
      });
      expect(set.status).toBe(503);
    } finally {
      await api(`/api/services/${serviceId}`, { method: "DELETE" });
    }
  });

  it("board config lifecycle: status + ticket-type create → rename → delete", async () => {
    const suffix = Date.now();

    const status = await api<{ id: string }>("/api/statuses", {
      method: "POST",
      ...json({ project_id: E2E_PROJECT_ID, name: `e2e-status-${suffix}` }),
    });
    expect(status.status).toBe(201);
    expect(
      (await api(`/api/statuses/${status.body!.id}`, {
        method: "PATCH",
        ...json({ name: `e2e-status-renamed-${suffix}`, kind: "progress" }),
      })).status,
    ).toBe(200);
    expect((await api(`/api/statuses/${status.body!.id}`, { method: "DELETE" })).status).toBe(204);

    const type = await api<{ id: string }>("/api/ticket-types", {
      method: "POST",
      ...json({ project_id: E2E_PROJECT_ID, name: `e2e-type-${suffix}` }),
    });
    expect(type.status).toBe(201);
    expect(
      (await api(`/api/ticket-types/${type.body!.id}`, {
        method: "PATCH",
        ...json({ name: `e2e-type-renamed-${suffix}` }),
      })).status,
    ).toBe(200);
    expect((await api(`/api/ticket-types/${type.body!.id}`, { method: "DELETE" })).status).toBe(204);
  });

  it("moves a ticket into a category and clears it again", async () => {
    const suffix = Date.now();
    const ticket = await api<{ id: string }>("/api/tickets", {
      method: "POST",
      ...json({ title: `e2e-move-${suffix}`, project_id: E2E_PROJECT_ID }),
    });
    expect(ticket.status).toBe(201);
    const category = await api<{ id: string }>("/api/categories", {
      method: "POST",
      ...json({ project_id: E2E_PROJECT_ID, name: `e2e-cat-${suffix}` }),
    });
    expect(category.status).toBe(201);

    try {
      expect(
        (await api(`/api/categories/${category.body!.id}/tickets/${ticket.body!.id}`, { method: "POST" }))
          .status,
      ).toBe(204);

      const moved = await api<{ category_id: string }>(`/api/tickets/${ticket.body!.id}`);
      expect(moved.body?.category_id).toBe(category.body!.id);

      expect(
        (await api(`/api/categories/tickets/${ticket.body!.id}`, { method: "DELETE" })).status,
      ).toBe(204);
      const cleared = await api<{ category_id: string }>(`/api/tickets/${ticket.body!.id}`);
      expect(cleared.body?.category_id).toBe("");
    } finally {
      await api(`/api/tickets/${ticket.body!.id}`, { method: "DELETE" });
      await api(`/api/categories/${category.body!.id}`, { method: "DELETE" });
    }
  });

  it("rollback fails with a not-found contract until a deploy exists", async () => {
    const svc = await api<{ id: string }>("/api/services", {
      method: "POST",
      ...json({
        project_id: E2E_PROJECT_ID,
        name: `e2e-rollback-${Date.now()}`,
        target: "e2e-host",
        strategy: "run",
        docker_network: "e2e-net",
        health_check: { url: "http://e2e/health" },
      }),
    });
    expect(svc.status).toBe(201);

    try {
      const rollback = await api<{ message?: string }>(`/api/services/${svc.body!.id}/rollback`, {
        method: "POST",
      });
      expect(rollback.status).toBe(404);
    } finally {
      await api(`/api/services/${svc.body!.id}`, { method: "DELETE" });
    }
  });

  it("audits authenticated gateway actions with the acting user", async () => {
    const ticket = await api<{ id: string }>("/api/tickets", {
      method: "POST",
      ...json({ title: `e2e-audit-${Date.now()}`, project_id: E2E_PROJECT_ID }),
    });
    expect(ticket.status).toBe(201);
    const ticketId = ticket.body!.id;

    try {
      expect(
        (await api(`/api/tickets/${ticketId}/status`, {
          method: "PATCH",
          ...json({ status: "in_progress" }),
        })).status,
      ).toBe(200);

      const action = `PATCH /api/tickets/${ticketId}/status`;
      let found = false;
      for (let attempt = 0; attempt < 10 && !found; attempt++) {
        const audit = await api<{ audit: { action: string; actor_type: string }[] }>(
          "/api/audit?limit=100",
        );
        expect(audit.status).toBe(200);
        found = (audit.body?.audit ?? []).some(
          (e) => e.action === action && e.actor_type === "user",
        );
        if (!found) await new Promise((resolve) => setTimeout(resolve, 300));
      }
      expect(found).toBe(true);
    } finally {
      await api(`/api/tickets/${ticketId}`, { method: "DELETE" });
    }
  });
});
