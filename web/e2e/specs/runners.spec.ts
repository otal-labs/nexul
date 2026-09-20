import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("runners (web)", () => {
  it("renders the runners and queue panels on a fresh instance", async () => {
    const page = await authedPage("/runners");
    // The RunnersPanel also has an h2 labelled "Runners" (the connected-runners sub-section) alongside the page's own h1, so scope to the page title.
    await expect(page.getByRole("heading", { level: 1, name: "Runners", exact: true })).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.getByRole("heading", { name: /Waiting for a runner/ })).toBeVisible();
    // the e2e runner service is always connected to the gateway
    await expect(page.getByText("e2e-runner", { exact: true })).toBeVisible({
      timeout: 15_000,
    });
  });
});
