import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { BASE_URL, E2E_WORKSPACE_PATH } from "../helpers";
import { anonPage, authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

describe("login / auth surface", () => {
  it("shows the home page with a sign-in CTA to unauthenticated visitors", async () => {
    const page = await anonPage("/");
    await page.waitForLoadState("networkidle");
    await expect(
      page.getByRole("heading", { name: /docs, tickets, chat, and deploys/i }),
    ).toBeVisible();
    await expect(page.getByRole("link", { name: /sign in/i })).toBeVisible();
  });

  it("serves a GitHub redirect at the OAuth start endpoint", async () => {
    const res = await fetch(`${BASE_URL}/auth/github`, { redirect: "manual" });
    expect(res.status).toBe(302);
    expect(res.headers.get("location")).toContain("github.com/login/oauth");
  });

  it("boots authenticated when a session is present", async () => {
    const page = await authedPage("/");
    await page.waitForLoadState("networkidle");
    expect(page.url()).toBe(`${BASE_URL}${E2E_WORKSPACE_PATH}`);
    await expect(
      page.getByRole("heading", { name: /docs, tickets, chat, and deploys/i }),
    ).toBeVisible();
    // Docs live in the per-project sidebar tree (ticket 13); there's no generic "Docs" link on the home page, so assert the logged-in CTA instead.
    await expect(page.getByRole("link", { name: /open board/i })).toBeVisible();
  });
});
