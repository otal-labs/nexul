import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_WORKSPACE_ID, makeApiClient, mintToken } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

// Projects have no standalone /projects list page (ticket 13); creation happens through the sidebar's "New project" dialog, and the result shows up as a row in the project tree.
describe("projects (web)", () => {
  it("creates a project from the sidebar and sees it in the tree", async () => {
    const name = `e2e-proj-ui-${Date.now()}`;
    const page = await authedPage("/");
    await page.waitForLoadState("networkidle");

    await page.getByRole("button", { name: "New project", exact: true }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Project name", { exact: true }).fill(name);
    await dialog.getByLabel("Prefix", { exact: true }).fill("EPJ");
    await dialog.getByRole("button", { name: /create project/i }).click();

    await expect(page.getByText(name, { exact: true })).toBeVisible({ timeout: 10_000 });

    const projects = await api<{ id: string; name: string }[]>(`/api/projects?workspace_id=${E2E_WORKSPACE_ID}`);
    const created = projects.body?.find((p) => p.name === name);
    if (created) await api(`/api/projects/${created.id}`, { method: "DELETE" });
  });
});
