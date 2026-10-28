import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_PROJECT_ID, E2E_WORKSPACE_PATH, makeApiClient, mintToken } from "../helpers";
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
});
