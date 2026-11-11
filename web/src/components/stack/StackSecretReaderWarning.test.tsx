import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StackSecretReaderWarning } from "@/components/stack/StackSecretReaderWarning";
import { DeployStrategy, type Stack } from "@/models/Stack";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "prod",
  strategy: DeployStrategy.Compose,
  env: { DB_PASSWORD: "op://vault/item/password", LOG_LEVEL: "info" },
  managed: true,
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
};

const runner = (name: string, machine: string, managers: { manager: string; state: string }[] = []) => ({
  id: name, name, machine, connected: true, last_seen: "2026-10-01T09:00:00Z", version: "", running_job: null, managers,
});

const renderWarning = async (runners: unknown[]) => {
  mocks.get.mockResolvedValue({ data: runners });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <StackSecretReaderWarning stack={stack} />
    </QueryClientProvider>,
  );
  await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/runners"));
  await vi.waitFor(() => expect(client.isFetching()).toBe(0));
};

describe("StackSecretReaderWarning", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it.each([
    ["a runner with no token", [runner("a", "prod")]],
    ["a runner whose token was rejected", [runner("a", "prod", [{ manager: "op", state: "token rejected" }]), runner("b", "qa", [{ manager: "op", state: "ready" }])]],
  ])("warns when the machine has only %s", async (_name, runners) => {
    await renderWarning(runners);

    expect(screen.getByRole("status")).toHaveTextContent("No runner on prod can read 1Password references yet");
    expect(screen.getByRole("status")).toHaveTextContent("Deploys will be refused until one has a token.");
  });

  it.each([
    ["a runner that is ready for the manager", [runner("a", "prod"), runner("b", "prod", [{ manager: "op", state: "ready" }])]],
    ["a runner that cannot be reached right now", [runner("a", "prod", [{ manager: "op", state: "unreachable" }])]],
    ["no runner the viewer can see on that machine", [runner("a", "qa")]],
  ])("says nothing with %s", async (_name, runners) => {
    await renderWarning(runners);

    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
