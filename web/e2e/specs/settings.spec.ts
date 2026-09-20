import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("settings (web)", () => {
  it("renders the default section and switches sections via the nav", async () => {
    const page = await authedPage("/settings");
    await page.waitForLoadState("networkidle");
    await expect(page.getByText("Instance URL").first()).toBeVisible();
    await page.getByRole("link", { name: "Tokens" }).click();
    await expect(page.getByText("Personal access tokens").first()).toBeVisible();
  });

  it("mints a personal access token from settings", async () => {
    const page = await authedPage("/settings?section=tokens");
    await page.waitForLoadState("networkidle");
    await page.getByLabel(/token name/i).fill("e2e-pat");
    await page.getByRole("button", { name: "Create token" }).click();
    await expect(page.getByText(/copy this token now/i)).toBeVisible();
  });
});
