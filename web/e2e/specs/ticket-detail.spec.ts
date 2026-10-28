import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_PROJECT_ID, E2E_WORKSPACE_ID, E2E_WORKSPACE_PATH, makeApiClient, mintToken } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

const json = <T = unknown>(body: T) => ({ body: JSON.stringify(body) });

const createTicket = async (title: string) => {
  const created = await api<{ id: string }>("/api/tickets", {
    method: "POST",
    ...json({ title, body: "walk the dog", project_id: E2E_PROJECT_ID }),
  });
  expect(created.status).toBe(201);
  return created.body!.id;
};

describe("ticket detail (web)", () => {
  it("shows the ticket and advances its status", async () => {
    const title = `e2e-ticket-detail-${Date.now()}`;
    const ticketId = await createTicket(title);

    try {
      const page = await authedPage(`${E2E_WORKSPACE_PATH}/tickets/${ticketId}`);
      await expect(page.getByRole("heading", { name: title })).toBeVisible({ timeout: 10_000 });
      await expect(page.getByRole("button", { name: "Status: Backlog" })).toBeVisible();

      // The status button is a Popover trigger listing the project's board columns.
      await page.getByRole("button", { name: "Status: Backlog" }).click();
      await page.getByRole("button", { name: "In progress", exact: true }).click();
      await expect(page.getByRole("button", { name: "Status: In progress" })).toBeVisible();

      // the transition persists after a reload
      await page.reload();
      await expect(page.getByRole("heading", { name: title })).toBeVisible();
      await expect(page.getByRole("button", { name: "Status: In progress" })).toBeVisible();
    } finally {
      await api(`/api/tickets/${ticketId}`, { method: "DELETE" });
    }
  });

  it("moves the ticket to another project", async () => {
    const title = `e2e-ticket-move-${Date.now()}`;
    const ticketId = await createTicket(title);

    const projectName = `e2e-proj-${Date.now()}`;
    const proj = await api<{ id: string }>("/api/projects", {
      method: "POST",
      ...json({ name: projectName, workspace_id: E2E_WORKSPACE_ID, prefix: "ETD" }),
    });
    expect(proj.status).toBe(201);
    const projectId = proj.body!.id;

    try {
      const page = await authedPage(`${E2E_WORKSPACE_PATH}/tickets/${ticketId}`);
      await expect(page.getByRole("heading", { name: title })).toBeVisible({ timeout: 10_000 });

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
