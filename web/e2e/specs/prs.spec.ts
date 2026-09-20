import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("pull requests (web)", () => {
  it("renders the search surface and fails closed on an unknown repo", async () => {
    const page = await authedPage("/prs");
    await expect(page.getByRole("heading", { name: "Pull requests", exact: true })).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.getByLabel("Owner")).toBeVisible();
    await expect(page.getByLabel("Repo")).toBeVisible();

    // An unknown repo deterministically fails (404 online, connection error offline); the feed must surface a clear error, never a blank page, and a GitHub-side failure must not sign the session out.
    await page.getByLabel("Owner").fill(`e2e-nonexistent-${Date.now()}`);
    await page.getByLabel("Repo").fill("no-such-repo");
    await page.getByRole("button", { name: /Load pull requests/i }).click();
    await expect(page.getByRole("alert")).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole("heading", { name: "Pull requests", exact: true })).toBeVisible();
  });
});
