import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_WORKSPACE_PATH } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("board / tickets (web)", () => {
  it("renders the board with the filter bar", async () => {
    const page = await authedPage(`${E2E_WORKSPACE_PATH}/board`);
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: "E2E Project", level: 1 })).toBeVisible();
    await page.getByRole("button", { name: /^Filter/ }).click();
    // The filter panel is a Radix Popover; its content portals outside <main>, so scoping to the main landmark never finds it. A project's own board is already scoped, so the panel has no Projects row.
    await expect(page.getByText("Status", { exact: true })).toBeVisible();
  });

  it("creates a ticket from the board", async () => {
    const title = `e2e-ticket-${Date.now()}`;
    const page = await authedPage(`${E2E_WORKSPACE_PATH}/board`);
    await page.waitForLoadState("networkidle");

    // Ticket creation is per-status-column ("New ticket in {status}"), not one page-level button; the seeded board's first column is Backlog.
    await page.getByRole("button", { name: "New ticket in Backlog" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(/title/i).fill(title);
    await dialog.getByRole("button", { name: /create/i }).click();

    await expect(page.getByText(title)).toBeVisible();
  });
});
