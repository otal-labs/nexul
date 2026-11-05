import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_WORKSPACE_PATH } from "../helpers";
import { closeBrowser, authedPage } from "./browser";

afterAll(closeBrowser);

const TITLE = `e2e-doc-${Date.now()}`;

describe("docs (web)", () => {
  it("shows the docs page and an empty state on a fresh instance", async () => {
    const page = await authedPage(`${E2E_WORKSPACE_PATH}/docs`);
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: /^Docs/, level: 1 })).toBeVisible();
  });

  it("creates a doc through the UI and sees it in the list", async () => {
    const page = await authedPage(`${E2E_WORKSPACE_PATH}/docs`);
    await page.waitForLoadState("networkidle");

    // The list pane's header carries an icon-only "New doc" too, so take the overview's.
    await page.getByRole("region", { name: "Docs overview" }).getByRole("button", { name: "New doc", exact: true }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(/title/i).fill(TITLE);
    await dialog.getByRole("button", { name: /create/i }).click();

    // The sidebar's project tree also lists every doc title directly (ticket 13), so scope to <main> to avoid colliding with that duplicate.
    await expect(page.getByRole("main").getByText(TITLE)).toBeVisible();
  });

  it("opens the doc detail view", async () => {
    const page = await authedPage(`${E2E_WORKSPACE_PATH}/docs`);
    await page.waitForLoadState("networkidle");
    // The list pane names the doc as well, so open it from the overview.
    await page.getByRole("region", { name: "Docs overview" }).getByText(TITLE, { exact: true }).click();
    await page.waitForLoadState("networkidle");
    // DocDetail renders the title as an editable Input once the collab session connects, not a static heading, since docs are inline-editable with no read-only state while live.
    await expect(page.getByLabel("Title", { exact: true })).toHaveValue(TITLE);
  });
});
