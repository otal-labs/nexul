import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { E2E_PROJECT_ID, makeApiClient, mintToken } from "../helpers";
import { authedPage, closeBrowser } from "./browser";

afterAll(closeBrowser);

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

const json = <T = unknown>(body: T) => ({ body: JSON.stringify(body) });

describe("service detail (web)", () => {
  it("shows the service settings and deploys an image from the page", async () => {
    const name = `e2e-svc-detail-${Date.now()}`;
    const created = await api<{ id: string }>("/api/services", {
      method: "POST",
      ...json({
        project_id: E2E_PROJECT_ID,
        name,
        target: "e2e-host",
        strategy: "run",
        docker_network: "e2e-net",
        health_check: { url: "http://e2e/health" },
      }),
    });
    expect(created.status).toBe(201);
    const serviceId = created.body!.id;

    try {
      const page = await authedPage(`/services/${serviceId}`);
      await expect(page.getByRole("heading", { name })).toBeVisible({ timeout: 10_000 });
      await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
      await expect(page.getByText("Target server")).toBeVisible();
      await expect(page.getByText("e2e-host", { exact: true })).toBeVisible();
      await expect(page.getByText("run strategy · e2e-host", { exact: true })).toBeVisible();

      // a fresh service has no healthy deploy, so rollback explains itself
      await expect(
        page.getByText(/Rollback needs at least one healthy deploy/),
      ).toBeVisible();

      const image = `e2e-img-${Date.now()}`;
      await page.getByLabel("Deploy image").fill(image);
      await page.getByRole("button", { name: /^Deploy$/ }).click();
      await expect(page.getByText("Deploy history")).toBeVisible();
      await expect(page.getByText(image, { exact: true })).toBeVisible({
        timeout: 10_000,
      });
    } finally {
      await api(`/api/services/${serviceId}`, { method: "DELETE" });
    }
  });
});
