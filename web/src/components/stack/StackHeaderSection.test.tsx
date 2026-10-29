import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StackHeaderSection } from "@/components/stack/StackHeaderSection";
import type { Stack } from "@/models/Stack";
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
  strategy: "run",
  managed: false,
  created_at: "2026-08-12T09:00:00Z",
  updated_at: "2026-08-12T09:00:00Z",
};

const renderHeader = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <StackHeaderSection stack={stack} latest={undefined} lastHealthy={undefined} image={undefined} hostnames={[]} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("StackHeaderSection Attach repository", () => {
  beforeEach(() => {
    resetViewer();
    viewer.showLocked = true;
  });

  it("is a link only for someone who can write both the stack and its project", async () => {
    viewer.workspace = ["stacks:write", "projects:write"];
    renderHeader();

    expect(await screen.findByRole("link", { name: "Attach repository" })).toHaveAttribute("href", "/acme/wizard/project/repository?stack=stack-1");
  });

  it("locks when the project can't be written, though the stack can", async () => {
    viewer.workspace = ["stacks:write"];
    const user = userEvent.setup();
    renderHeader();

    const attach = await screen.findByRole("button", { name: "Attach repository" });
    await user.click(attach);

    expect(attach).toHaveAttribute("aria-disabled", "true");
    expect(attach).toHaveAccessibleDescription("Your role can't attach repositories");
    expect(screen.queryByRole("link", { name: "Attach repository" })).not.toBeInTheDocument();
  });
});
