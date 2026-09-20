import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("members (web)", () => {
  it("shows the invitation-link management surface", async () => {
    const page = await authedPage("/members");
    await expect(page.getByRole("heading", { name: "Members", exact: true })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole("button", { name: "Invite", exact: true })).toBeVisible();
    await expect(page.getByText(/active invitation links/i)).toBeVisible();
  });
});
