import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { BASE_URL, E2E_PROJECT_ID, mintToken } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("live updates (WebSocket push)", () => {
  it("shows a new ticket on the board without a reload", async () => {
    const title = `live-${Date.now()}`;
    const page = await authedPage("/board");
    await page.waitForLoadState("networkidle");
    await expect(page.getByText(title)).not.toBeVisible();

    // Creates the ticket through the API; the UI must learn about it via the /ws/events push + query invalidation, not a page reload.
    const res = await fetch(`${BASE_URL}/api/tickets`, {
      method: "POST",
      headers: { Authorization: `Bearer ${mintToken()}`, "Content-Type": "application/json" },
      body: JSON.stringify({ title, project_id: E2E_PROJECT_ID }),
    });
    expect(res.status).toBe(201);

    await expect(page.getByText(title)).toBeVisible({ timeout: 15_000 });
  });
});
