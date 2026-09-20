import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { makeApiClient, mintToken } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

describe("members (web)", () => {
  // The instance-wide sign-in allowlist (AU4) lives under Settings' "Instance access" section; the top-level Members page is the workspace-scoped roster/invite screen (ADR 0024).
  it("allows and removes a sign-in allowlist entry from Settings", async () => {
    const login = `e2e-member-${Date.now()}`;
    const page = await authedPage("/settings?section=access");
    await expect(page.getByRole("heading", { name: "Instance access", exact: true })).toBeVisible({
      timeout: 10_000,
    });

    await page.getByLabel("GitHub username").fill(login);
    await page.getByRole("button", { name: "Allow", exact: true }).click();
    await expect(page.getByText(login, { exact: true })).toBeVisible({ timeout: 10_000 });

    await page.getByRole("button", { name: `Remove ${login} from the allowlist` }).click();
    await expect(page.getByText(login, { exact: true })).toHaveCount(0);

    const members = await api<{ members: string[] }>("/api/auth/members");
    expect(members.body?.members).not.toContain(login);
  });
});
