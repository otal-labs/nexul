import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("settings (web)", () => {
  it("renders the default section and switches sections via the nav", async () => {
    const page = await authedPage("/settings");
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: "Profile", level: 1 })).toBeVisible();
    await page.getByRole("link", { name: "Instance", exact: true }).click();
    await expect(page.getByText("Instance URL").first()).toBeVisible();
    await page.getByRole("link", { name: "Security", exact: true }).click();
    await expect(page.getByText("Personal access tokens").first()).toBeVisible();
  });

  it("mints a personal access token from settings", async () => {
    const page = await authedPage("/settings/security");
    await page.waitForLoadState("networkidle");
    await page.getByLabel(/token name/i).fill("e2e-pat");
    await page.getByRole("button", { name: "Create token" }).click();
    await expect(page.getByText(/copy this token now/i)).toBeVisible();
  });
});
