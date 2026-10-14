import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DeployRetryButton } from "@/components/deploy/DeployRetryButton";
import { useFetchStack } from "@/hooks/StackHooks";
import { DeployStatus, DeployStrategy, type Deploy, type Stack } from "@/models/Stack";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), areas: ["editDeploys"] as string[] }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: mocks.post }, errorMessage: () => "boom" }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => mocks.areas.includes(area) }));

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "instance",
  strategy: DeployStrategy.Compose,
  managed: true,
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
};

const failed: Deploy = {
  id: "d-1",
  kind: "build",
  stack_id: "stack-1",
  service: "api",
  target: "instance",
  image: "",
  status: DeployStatus.Failed,
  strategy: DeployStrategy.Compose,
  ref: "main",
  commit: "9f2c1ab4e7d0",
  created_at: "2026-10-01T10:00:00Z",
  updated_at: "2026-10-01T10:01:00Z",
};

const Where = () => <p data-testid="where">{useLocation().pathname}</p>;

// Shows once the stack the button reads has loaded, so a missing button is the button's answer and not a pending fetch.
const StackLoaded = () => (useFetchStack("stack-1").data ? <p>stack loaded</p> : null);

const renderButton = (deploy: Deploy) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <DeployRetryButton deploy={deploy} />
        <StackLoaded />
        <Where />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("DeployRetryButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.areas = ["editDeploys"];
    mocks.get.mockResolvedValue({ data: stack });
  });

  it("starts a failed deploy again and opens the new one", async () => {
    mocks.post.mockResolvedValue({ data: { ...failed, id: "d-2", status: DeployStatus.Pending } });
    const user = userEvent.setup();
    renderButton(failed);

    await user.click(await screen.findByRole("button", { name: "Retry" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/deploys/d-1/retry");
    expect(await screen.findByText("/acme/stacks/stack-1/deploys/d-2")).toBeInTheDocument();
  });

  it.each([
    ["a deploy that is still going", { status: DeployStatus.Running }],
    ["a deploy that came up healthy", { status: DeployStatus.Healthy }],
    ["a failed deploy that recorded nothing to rebuild", { ref: "", commit: "", image: "" }],
  ])("offers no Retry on %s", async (_name, change) => {
    renderButton({ ...failed, ...change });
    await screen.findByText("stack loaded");
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("offers no Retry to someone who cannot deploy", async () => {
    mocks.areas = [];
    renderButton(failed);
    await screen.findByText("stack loaded");
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });
});
