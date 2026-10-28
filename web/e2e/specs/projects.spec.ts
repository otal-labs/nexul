import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_WORKSPACE_ID, E2E_WORKSPACE_PATH, makeApiClient, mintToken } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

// Projects have no standalone list page; creation goes through the sidebar switcher into the project wizard, whose Name step creates the project, and the result shows up in the switcher.
describe("projects (web)", () => {
  it("creates a project from the sidebar and sees it in the tree", async () => {
    const name = `e2e-proj-ui-${Date.now()}`;
    const page = await authedPage(E2E_WORKSPACE_PATH);
    await page.waitForLoadState("networkidle");

    await page.getByRole("button", { name: "E2E Project", exact: true }).click();
    await page.getByRole("dialog").getByRole("button", { name: "New project", exact: true }).click();
    await page.getByLabel("Project name", { exact: true }).fill(name);
    await page.getByLabel("Prefix", { exact: true }).fill("EPJ");
    await page.getByRole("button", { name: "Continue", exact: true }).click();
    await expect(page.getByText("Project created")).toBeVisible({ timeout: 10_000 });

    await page.getByRole("button", { name: "Finish later", exact: true }).click();
    await expect(page.getByRole("heading", { name, level: 1 })).toBeVisible({ timeout: 10_000 });
    await page.getByRole("button", { name, exact: true }).click();
    await expect(page.getByRole("dialog").getByRole("button", { name, exact: true })).toBeVisible();

    const projects = await api<{ id: string; name: string }[]>(`/api/projects?workspace_id=${E2E_WORKSPACE_ID}`);
    const created = projects.body?.find((p) => p.name === name);
    if (created) await api(`/api/projects/${created.id}`, { method: "DELETE" });
  });
});
