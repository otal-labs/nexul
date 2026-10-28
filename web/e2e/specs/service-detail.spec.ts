import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_MACHINE, E2E_PROJECT_ID, E2E_WORKSPACE_PATH, makeApiClient, mintToken } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

const json = <T = unknown>(body: T) => ({ body: JSON.stringify(body) });

describe("service detail (web)", () => {
  it("shows the service settings and its deploy history", async () => {
    const name = `e2e-svc-detail-${Date.now()}`;
    const created = await api<{ id: string }>("/api/stacks", {
      method: "POST",
      ...json({
        project_id: E2E_PROJECT_ID,
        name,
        machine: E2E_MACHINE,
        strategy: "run",
        docker_network: "e2e-net",
      }),
    });
    expect(created.status).toBe(201);
    const serviceId = created.body!.id;

    try {
      const page = await authedPage(`${E2E_WORKSPACE_PATH}/stacks/${serviceId}`);
      await expect(page.getByRole("heading", { name, level: 1 })).toBeVisible({ timeout: 10_000 });
      await expect(page.getByRole("heading", { name: "Deploy", level: 2 })).toBeVisible();
      await expect(page.getByTitle("Runner", { exact: true })).toHaveText(E2E_MACHINE);
      await expect(page.getByTitle("Strategy", { exact: true })).toHaveText("run");

      // a fresh stack has no repository and nothing running, so the page offers no deploy and no rollback and says why
      await expect(
        page.getByText(/no repository to build from and nothing running to redeploy/),
      ).toBeVisible();
      await expect(page.getByRole("button", { name: "Rollback" })).toHaveCount(0);

      // the page no longer deploys an image itself; a deploy requested elsewhere shows up in its history
      const image = `e2e-img-${Date.now()}`;
      const deploy = await api("/api/deploys", { method: "POST", ...json({ service_id: serviceId, image }) });
      expect(deploy.status).toBe(201);
      await page.getByRole("link", { name: "Deploy history", exact: true }).click();
      await expect(page.getByRole("heading", { name: "Deploy history", level: 2 })).toBeVisible();
      await expect(page.getByRole("region", { name: "Deploy history" }).getByText(image, { exact: true })).toBeVisible({
        timeout: 10_000,
      });
    } finally {
      await api(`/api/stacks/${serviceId}`, { method: "DELETE" });
    }
  });
});
