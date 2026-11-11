import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StackSecretsChangedNotice } from "@/components/stack/StackSecretsChangedNotice";
import { DeployStrategy, type Stack } from "@/models/Stack";
import { resetViewer, viewer } from "@/test/viewerActions";

vi.mock("@/api/client", async () => {
  const { withOwner } = await import("@/test/viewer");
  return { api: { get: withOwner(() => Promise.resolve({ data: [] })) }, errorMessage: () => "boom" };
});
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => () => false }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "instance",
  strategy: DeployStrategy.Run,
  managed: true,
  created_at: "2026-08-12T09:00:00Z",
  updated_at: "2026-08-12T09:00:00Z",
};

const renderNotice = (listed: Stack = stack) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <StackSecretsChangedNotice stack={listed} image="ghcr.io/acme/api:v1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("StackSecretsChangedNotice", () => {
  beforeEach(() => {
    resetViewer();
    viewer.showLocked = false;
  });

  it("names the changed variables to a viewer who cannot deploy, with no Redeploy", async () => {
    viewer.workspace = ["stacks:read"];
    renderNotice({ ...stack, secrets_changed: ["DB_PASSWORD", "API_KEY"] });

    expect(await screen.findByRole("status")).toHaveTextContent("DB_PASSWORD, API_KEY changed since the last deploy");
    expect(screen.queryByRole("button", { name: "Redeploy" })).not.toBeInTheDocument();
  });

  it("offers Redeploy to a viewer who may deploy, and reads naturally for one name", async () => {
    viewer.workspace = ["deploys:write"];
    renderNotice({ ...stack, secrets_changed: ["DB_PASSWORD"] });

    expect(await screen.findByRole("status")).toHaveTextContent("DB_PASSWORD changed since the last deploy");
    expect(await screen.findByRole("button", { name: "Redeploy" })).toBeEnabled();
  });

  it.each([[stack], [{ ...stack, secrets_changed: [] }]])("says nothing when the stack lists no changed variable (%j)", (listed) => {
    renderNotice(listed);

    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
