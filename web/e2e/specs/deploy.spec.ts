import { describe, expect, it } from "vitest";

import { E2E_PROJECT_ID, makeApiClient, mintToken } from "../helpers";

const TOKEN = mintToken();
const api = makeApiClient(TOKEN);

const json = <T = unknown>(body: T) => ({ body: JSON.stringify(body) });

type Deploy = { id: string; status: string; image: string };

const waitForDeploy = async (deployId: string, statuses: string[], timeoutMs: number) => {
  const deadline = Date.now() + timeoutMs;
  let deploy: Deploy | null = null;
  while (Date.now() < deadline) {
    const got = await api<Deploy>(`/api/deploys/${deployId}`);
    deploy = got.body;
    if (got.status === 200 && deploy && statuses.includes(deploy.status)) break;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  return deploy;
};

describe("deploy execution (real runner)", () => {
  it(
    "runs a deploy on the runner, serves it on the host, and rolls it back",
    async () => {
      const name = `e2e-deploy-${Date.now()}`;
      const svc = await api<{ id: string }>("/api/services", {
        method: "POST",
        ...json({
          project_id: E2E_PROJECT_ID,
          name,
          target: "e2e-host",
          strategy: "run",
          docker_network: "e2e-net",
          health_check: { url: `http://${name}` },
        }),
      });
      expect(svc.status).toBe(201);
      const serviceId = svc.body!.id;

      try {
        // The deploy request is queued server-side until the e2e runner (connected to the runner gateway) claims it.
        const dep = await api<Deploy>("/api/deploys", {
          method: "POST",
          ...json({ service_id: serviceId, image: "nginx:alpine" }),
        });
        expect(dep.status).toBe(201);
        const deployId = dep.body!.id;

        const healthy = await waitForDeploy(deployId, ["healthy", "failed"], 90_000);
        expect(healthy?.status).toBe("healthy");

        // The deployed container joined the e2e-net docker network, so its container name resolves from the spec.
        let served = false;
        for (let attempt = 0; attempt < 15 && !served; attempt++) {
          try {
            const res = await fetch(`http://${name}/`);
            served = res.status === 200 && (await res.text()).includes("Welcome to nginx");
          } catch {
            await new Promise((resolve) => setTimeout(resolve, 1000));
          }
        }
        expect(served).toBe(true);

        // Rollback enqueues a fresh deploy of the last healthy image and the runner runs it.
        const rollback = await api<Deploy>(`/api/services/${serviceId}/rollback`, {
          method: "POST",
        });
        expect(rollback.status).toBe(201);
        const rolled = await waitForDeploy(rollback.body!.id, ["healthy", "failed"], 90_000);
        expect(rolled?.status).toBe("healthy");

        const runners = await api<{ id: string; name: string }[]>("/api/runners");
        expect(runners.status).toBe(200);
        expect(runners.body?.some((r) => r.name === "e2e-runner")).toBe(true);
      } finally {
        await api(`/api/services/${serviceId}`, { method: "DELETE" });
      }
    },
    180_000,
  );
});
