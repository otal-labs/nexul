import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "refused" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const catalog = [
  { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "project" },
  { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write", area: "project" },
];

const roles = [
  { id: "r-owner", name: "Owner", permissions: [], is_owner_role: true, description: "Runs the workspace." },
  { id: "r-client", name: "Client", permissions: ["tickets:read"], is_owner_role: false, description: "Follows along and reads tickets." },
  { id: "r-dev", name: "Developer", permissions: ["tickets:read", "tickets:write"], is_owner_role: false, description: "" },
];

describe("CreateInvitationForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/workspaces") return Promise.resolve({ data: [{ id: "ws-1", name: "Acme", slug: "acme" }] });
      if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: roles });
      if (url === "/api/workspaces/ws-1/me") return Promise.resolve({ data: { role_name: "Owner", roles: [{ id: "r-owner", name: "Owner", is_owner: true }], permissions: [] } });
      if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
      return Promise.resolve({ data: [] });
    });
  });

  it("offers Remove on a workspace only once the link admits to two, so it always admits somewhere", async () => {
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <CreateInvitationDialog onCreated={vi.fn()} />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Invite" }));
    const dialog = within(await screen.findByRole("dialog"));
    expect(await dialog.findByText("Roles in Acme")).toBeInTheDocument();
    expect(dialog.queryByRole("button", { name: /^Remove workspace/ })).not.toBeInTheDocument();

    await user.click(dialog.getByRole("button", { name: "Add workspace" }));
    await user.click(dialog.getByRole("button", { name: "Remove workspace 2" }));
    expect(dialog.queryByRole("button", { name: /^Remove workspace/ })).not.toBeInTheDocument();

    // An open dialog stays in the shared confirmation root into the next case.
    await user.keyboard("{Escape}");
    await vi.waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("holds Create link until a role is ticked, then sends every role ticked", async () => {
    mocks.post.mockResolvedValue({ data: { id: "inv-1", url: "https://nexul.example.com/invite#t", grants: [] } });
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <CreateInvitationDialog onCreated={vi.fn()} />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Invite" }));
    expect(await screen.findByText("Roles in Acme")).toBeInTheDocument();
    expect(await screen.findByText("Pick at least one role. An invitation gives a role on joining.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create link" })).toBeDisabled();

    await user.click(screen.getByRole("checkbox", { name: /Client/ }));
    await user.click(screen.getByRole("checkbox", { name: /Developer/ }));
    await user.click(screen.getByRole("button", { name: "Create link" }));

    await vi.waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/invitations", { expires_in_days: 7, grants: [{ workspace_id: "ws-1", role_ids: ["r-client", "r-dev"] }] }),
    );
  });

  it("explains who the link is for and shows each role's description under its name, then its reach as its page words it", async () => {
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <CreateInvitationDialog onCreated={vi.fn()} />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Invite" }));
    expect(await screen.findByText("Roles in Acme")).toBeInTheDocument();

    expect(
      screen.getByText("For someone without an account. The link works once."),
    ).toBeInTheDocument();
    const labelOf = (name: string) => screen.getByText(name).closest("label") as HTMLElement;
    expect(within(labelOf("Client")).getByText("Follows along and reads tickets.")).toBeInTheDocument();
    expect(within(labelOf("Owner")).getByText("Runs the workspace.")).toBeInTheDocument();
    expect(within(labelOf("Client")).getByText("In projects: 1 of 1, up to Read")).toBeInTheDocument();
    expect(within(labelOf("Developer")).getByText("In projects: 1 of 1, up to Write")).toBeInTheDocument();
  });
});
