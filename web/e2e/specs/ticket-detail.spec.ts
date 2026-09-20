import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_PROJECT_ID, E2E_WORKSPACE_ID, makeApiClient, mintToken } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

const json = <T = unknown>(body: T) => ({ body: JSON.stringify(body) });

describe("ticket detail (web)", () => {
  it("shows the ticket, advances its status, and moves it to another project", async () => {
    const title = `e2e-ticket-detail-${Date.now()}`;
    const created = await api<{ id: string }>("/api/tickets", {
      method: "POST",
      ...json({ title, body: "walk the dog", project_id: E2E_PROJECT_ID }),
    });
    expect(created.status).toBe(201);
    const ticketId = created.body!.id;

    const projectName = `e2e-proj-${Date.now()}`;
    const proj = await api<{ id: string }>("/api/projects", {
      method: "POST",
      ...json({ name: projectName, workspace_id: E2E_WORKSPACE_ID, prefix: "ETD" }),
    });
    expect(proj.status).toBe(201);
    const projectId = proj.body!.id;

    try {
      // TicketPropertiesPanel renders TicketStatusBadge twice (a plain badge plus an actionable status control), both showing the same text, so scope to .first() rather than a plain exact-text match.
      const page = await authedPage(`/tickets/${ticketId}`);
      await expect(page.getByRole("heading", { name: title })).toBeVisible({ timeout: 10_000 });
      await expect(page.getByText("open", { exact: true }).first()).toBeVisible();

      // The status badge is a Popover trigger; StatusTransitionButtons only renders once it's opened.
      await page.getByRole("button", { name: "open" }).click();
      await page.getByRole("button", { name: "Move to In progress" }).click();
      await expect(page.getByText("in progress", { exact: true }).first()).toBeVisible();

      // the transition persists after a reload
      await page.reload();
      await expect(page.getByRole("heading", { name: title })).toBeVisible();
      await expect(page.getByText("in progress", { exact: true }).first()).toBeVisible();

      // exact: true because the sidebar's "New doc in {project}" buttons also carry "Project" in their aria-label (ticket 13).
      await page.getByLabel("Project", { exact: true }).selectOption({ label: projectName });
      await expect(page.getByLabel("Project", { exact: true })).toHaveValue(projectId);
      await page.reload();
      await expect(page.getByRole("heading", { name: title })).toBeVisible();
      await expect(page.getByLabel("Project", { exact: true })).toHaveValue(projectId);
    } finally {
      await api(`/api/tickets/${ticketId}`, { method: "DELETE" });
      await api(`/api/projects/${projectId}`, { method: "DELETE" });
    }
  });
});
