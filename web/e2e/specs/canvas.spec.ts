import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("topology canvas (drawing the infrastructure map)", () => {
  it("adds an external node through the palette", async () => {
    const name = `proxy-${Date.now()}`;
    const page = await authedPage("/topology");
    await page.waitForLoadState("networkidle");

    await page.getByRole("button", { name: /external/i }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Name", { exact: true }).fill(name);
    await dialog.getByRole("button", { name: /add/i }).click();

    await expect(page.locator(".react-flow").getByText(name)).toBeVisible();
  });

  it("adds a network node through the palette", async () => {
    const name = `net-${Date.now()}`;
    const page = await authedPage("/topology");
    await page.waitForLoadState("networkidle");

    await page.getByRole("button", { name: /network/i }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(/network name/i).fill(name);
    await dialog.getByRole("button", { name: /add/i }).click();

    await expect(page.locator(".react-flow").getByText(name)).toBeVisible();
  });
});
