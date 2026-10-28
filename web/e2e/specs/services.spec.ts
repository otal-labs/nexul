import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_PROJECT_ID, E2E_WORKSPACE_PATH } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

// Services live in the per-project settings page (ticket 13); there's no standalone /services list or project picker, ProjectServices takes the project id straight from the route.
describe("services (web)", () => {
  it("renders the project's services section", async () => {
    const page = await authedPage(`${E2E_WORKSPACE_PATH}/projects/${E2E_PROJECT_ID}/settings/services`);
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: "Services", level: 2 })).toBeVisible();
    await expect(page.getByRole("link", { name: /new service/i })).toBeVisible();
  });
});
