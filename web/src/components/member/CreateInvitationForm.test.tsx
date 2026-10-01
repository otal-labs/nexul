import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "refused" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const catalog = [
  { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "project" },
  { value: "chat:read", label: "Read chat", domain: "chat", action: "read", area: "workspace" },
];

describe("CreateInvitationForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/workspaces") return Promise.resolve({ data: [{ id: "ws-1", name: "OTAL", slug: "otal" }] });
      if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: [{ id: "r-client", name: "Client", permissions: [], is_owner_role: false }] });
      if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
      if (url === "/api/projects") return Promise.resolve({ data: [{ id: "p-web", name: "Web", prefix: "WEB" }, { id: "p-api", name: "Api", prefix: "API" }] });
      return Promise.resolve({ data: [] });
    });
  });

  it("builds a link that lands the person held to the projects chosen", async () => {
    mocks.post.mockResolvedValue({ data: { id: "inv-1", url: "https://nexul.test/invite#t", grants: [] } });
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <CreateInvitationDialog />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Invite" }));
    await user.click(await screen.findByRole("combobox", { name: "Every project for the invited person" }));
    await user.click(await screen.findByRole("option", { name: "Only chosen projects" }));
    expect(await screen.findByText("Projects · 0 of 2")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Web access: None" }));
    await user.click(await screen.findByRole("menuitemradio", { name: /^Read/ }));
    await user.click(screen.getByRole("button", { name: "Create link" }));

    await vi.waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith(
        "/api/invitations",
        expect.objectContaining({
          grants: [expect.objectContaining({ workspace_id: "ws-1", role_id: "r-client", every_project: "none", project_access: [{ project_id: "p-web", allow: ["tickets:read"] }] })],
        }),
      ),
    );
  });
});
