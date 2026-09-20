import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("topology (web)", () => {
  it("renders the React Flow canvas", async () => {
    const page = await authedPage("/topology");
    await page.waitForLoadState("networkidle");
    await expect(page.locator(".react-flow")).toBeVisible();
  });
});
