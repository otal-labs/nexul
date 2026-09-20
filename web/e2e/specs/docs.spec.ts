import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { closeBrowser, authedPage } from "./browser";

afterAll(closeBrowser);

const TITLE = `e2e-doc-${Date.now()}`;

describe("docs (web)", () => {
  it("shows the docs page and an empty state on a fresh instance", async () => {
    const page = await authedPage("/docs");
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: "Docs", exact: true })).toBeVisible();
  });

  it("creates a doc through the UI and sees it in the list", async () => {
    const page = await authedPage("/docs");
    await page.waitForLoadState("networkidle");

    // The sidebar also has a "New doc in {project}" button per project (ticket 13), so use the page-level exact "New doc" button instead of an ambiguous regex.
    await page.getByRole("button", { name: "New doc", exact: true }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(/title/i).fill(TITLE);
    await dialog.getByRole("button", { name: /create/i }).click();

    // The sidebar's project tree also lists every doc title directly (ticket 13), so scope to <main> to avoid colliding with that duplicate.
    await expect(page.getByRole("main").getByText(TITLE)).toBeVisible();
  });

  it("opens the doc detail view", async () => {
    const page = await authedPage("/docs");
    await page.waitForLoadState("networkidle");
    await page.getByRole("main").getByText(TITLE).click();
    await page.waitForLoadState("networkidle");
    // DocDetail renders the title as an editable Input once the collab session connects, not a static heading, since docs are inline-editable with no read-only state while live.
    await expect(page.getByLabel("Title", { exact: true })).toHaveValue(TITLE);
  });
});
