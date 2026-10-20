import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardPeopleStep } from "@/components/wizard/WizardPeopleStep";
import type { Project } from "@/models/Project";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }));
vi.mock("@/api/client", async () => {
  const { withOwner } = await import("@/test/viewer");
  return { api: { get: withOwner(mocks.get), put: mocks.put }, errorMessage: () => "refused" };
});
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const project = { id: "p-1", name: "Storefront", workspace_id: "ws-1" } as Project;
const role = (id: string, name: string, owner = false) => ({ id, workspace_id: "ws-1", name, permissions: [], is_owner_role: owner, created_at: "", updated_at: "" });
// Creating the project listed its creator and the creator's one role, Developer.
const grants = [
  { resource_type: "project", resource_id: "p-1", target: "person", user_id: "alice", allow: ["projects:access", "projects:open"], deny: [], created_at: "", updated_at: "" },
  { resource_type: "project", resource_id: "p-1", target: "role", user_id: "", role_id: "r-dev", allow: ["projects:open"], deny: [], created_at: "", updated_at: "" },
];

describe("WizardPeopleStep", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.put.mockResolvedValue({ data: null });
    mocks.get.mockImplementation((url: string) => {
      if (url.startsWith("/api/permissions?")) return Promise.resolve({ data: { grants } });
      if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: [role("r-owner", "Owner", true), role("r-dev", "Developer"), role("r-client", "Client")] });
      if (url === "/api/workspaces/ws-1/people") return Promise.resolve({ data: { people: [] } });
      return Promise.reject(new Error(`unexpected GET ${url}`));
    });
  });

  it("starts with the creator's roles ticked, and saves only what changed before moving on", async () => {
    const user = userEvent.setup();
    const onDone = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <WizardPeopleStep project={project} onDone={onDone} onBack={vi.fn()} onSkip={vi.fn()} />
      </QueryClientProvider>,
    );

    expect(await screen.findByRole("checkbox", { name: "Developer" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Client" })).not.toBeChecked();
    expect(screen.queryByRole("checkbox", { name: "Owner" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Client" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.put).toHaveBeenCalledTimes(1);
    expect(mocks.put).toHaveBeenCalledWith("/api/permissions", { resource_type: "project", resource_ids: ["p-1"], role_ids: ["r-client"], actions: ["projects:open"], state: "allow" });
  });
});
