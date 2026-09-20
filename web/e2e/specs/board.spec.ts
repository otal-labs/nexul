import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("board / tickets (web)", () => {
  it("renders the board with the filter bar", async () => {
    const page = await authedPage("/board");
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: /board/i })).toBeVisible();
    await page.getByRole("button", { name: /^Filter/ }).click();
    // The filter panel is a Radix Popover; its content portals outside <main>, so scoping to the main landmark never finds it.
    await expect(page.getByText("Projects", { exact: true })).toBeVisible();
  });

  it("creates a ticket from the board", async () => {
    const title = `e2e-ticket-${Date.now()}`;
    const page = await authedPage("/board");
    await page.waitForLoadState("networkidle");

    // Ticket creation is per-status-column ("New ticket in {status}"), not one page-level button; the seeded board's first column is Open.
    await page.getByRole("button", { name: "New ticket in Open" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(/title/i).fill(title);
    await dialog.getByRole("button", { name: /create/i }).click();

    await expect(page.getByText(title)).toBeVisible();
  });
});
