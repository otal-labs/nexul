import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_PROJECT_ID } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

// Services live in the per-project settings page (ticket 13); there's no standalone /services list or project picker, ProjectServices takes the project id straight from the route.
describe("services (web)", () => {
  it("renders the project's services section", async () => {
    const page = await authedPage(`/projects/${E2E_PROJECT_ID}/settings`);
    await page.waitForLoadState("networkidle");
    await expect(page.getByText("Services", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: /new service/i })).toBeVisible();
  });

  it("creates a service for the seeded project through the dialog", async () => {
    const name = `e2e-svc-${Date.now()}`;
    const page = await authedPage(`/projects/${E2E_PROJECT_ID}/settings`);
    await page.waitForLoadState("networkidle");

    await page.getByRole("button", { name: /new service/i }).click();

    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Name", { exact: true }).fill(name);
    await dialog.getByLabel("Target server", { exact: true }).fill("e2e-host");
    await dialog.getByLabel("Compose directory", { exact: true }).fill("/srv/app");
    await dialog.getByLabel(/image/i).fill("alpine");
    await dialog.getByLabel(/health check url/i).fill("http://e2e/health");
    // The dialog's OK button sits below the fold in a scroll-fighting container, so focus it and press Enter instead of clicking.
    const ok = dialog.getByRole("button", { name: /create/i });
    await ok.focus();
    await ok.press("Enter");

    await expect(page.getByText(name)).toBeVisible();
  });
});
