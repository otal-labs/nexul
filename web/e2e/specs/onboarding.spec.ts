import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { authedPage, closeBrowser } from "./browser";
import { clearDefaultWorkspaceOwnership, restoreDefaultWorkspaceOwnership, updateUser } from "../setup/db";

// The wizards restore their flags on completion; afterAll is a safety net so a mid-test failure can't leave the seeded user stranded in a wizard, or leave the default workspace without an Owner role for every other spec file that assumes one exists.
afterAll(() => {
  updateUser({ can_create_workspace: 1, first_login_done: 1 });
  restoreDefaultWorkspaceOwnership();
  return closeBrowser();
});

describe("onboarding wizards", () => {
  it("first-login wizard: confirms and continues into the app", async () => {
    updateUser({ first_login_done: 0 });
    const page = await authedPage("/docs");
    await page.waitForURL("**/onboarding/profile");
    await expect(page.getByRole("heading", { name: /welcome/i })).toBeVisible();
    await page.getByRole("button", { name: "Continue", exact: true }).click();
    await page.waitForURL("**/");
    await expect(page.getByRole("heading", { name: /docs become tickets/i })).toBeVisible();
  });

  it("owner wizard: first user sets up their workspace and connects tools", async () => {
    updateUser({ can_create_workspace: 0, first_login_done: 1 });
    // global.ts's seed pre-creates the default workspace's Owner role/membership for every other spec's benefit, but tenancy.BindDefaultWorkspaceOwner always INSERTs a fresh one (idx_roles_owner_singleton), so this genuinely-fresh-owner flow 409s unless that pre-seeded row is cleared first.
    clearDefaultWorkspaceOwnership();
    const page = await authedPage("/");
    await page.waitForURL("**/onboarding/owner");

    await page.getByRole("button", { name: "Continue", exact: true }).click();

    // The seeded default project starts with an empty prefix (ADR 0004 backfill), which fails the 2-5 letter validation, so it must be filled before continuing.
    await expect(page.getByLabel("Workspace name", { exact: true })).toBeVisible();
    await page.getByLabel("Project prefix", { exact: true }).fill("GEN");
    await page.getByRole("button", { name: "Continue", exact: true }).click();

    await page.getByRole("button", { name: /finish setup/i }).click();
    // completion flows into the optional DNS step (DN3)
    await page.waitForURL("**/onboarding/dns");
    await expect(
      page.getByRole("heading", { name: /connect your dns provider/i }),
    ).toBeVisible();
  });

  it("dns onboarding renders and can be skipped", async () => {
    const page = await authedPage("/onboarding/dns");
    await page.waitForLoadState("networkidle");
    await expect(
      page.getByRole("heading", { name: /connect your dns provider/i }),
    ).toBeVisible();
    await page.getByRole("button", { name: /skip for now/i }).click();
    await page.waitForURL("**/");
  });
});
