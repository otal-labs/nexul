import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { AccountMenu } from "@/components/AccountMenu";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const ownerUser = {
  id: "u1",
  provider: "github" as const,
  provider_user_id: "1",
  login: "onik97",
  name: "Onik",
  avatar_url: "",
  can_create_workspace: true,
  first_login_done: true,
  created_at: "2026-08-12T12:00:00Z",
};

let roleResponse: { role_name: string; permissions?: string[] } = {
  role_name: "Owner",
  permissions: ["projects:write"],
};

const renderMenu = (collapsed = false) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AccountMenu collapsed={collapsed} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("AccountMenu", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.mocked(api.get).mockReset();
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
    roleResponse = { role_name: "Owner", permissions: ["projects:write"] };
    // /api/auth/me returns the user; every other GET returns the workspace
    // role. Override the role via roleResponse, not by replacing this dispatcher.
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") {
        return { data: { user: ownerUser, needs_owner_wizard: false, needs_first_login_wizard: false } };
      }
      return { data: roleResponse };
    });
  });

  it("renders nothing while the profile hasn't loaded", () => {
    vi.mocked(api.get).mockImplementation(async () => new Promise(() => {}));
    const { container } = renderMenu();
    expect(container).toBeEmptyDOMElement();
  });

  it("opens the popover with Settings, Support, and Logout", async () => {
    const user = userEvent.setup();
    renderMenu();
    await user.click(await screen.findByText("@onik97"));

    expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/settings");
    const support = screen.getByRole("link", { name: "Support" });
    expect(support).toHaveAttribute("href", "https://github.com/otal-labs/nexul/issues");
    expect(support).toHaveAttribute("target", "_blank");
    expect(screen.getByRole("button", { name: "Logout" })).toBeInTheDocument();
  });

  it("hides Settings for a member without projects:write", async () => {
    roleResponse = { role_name: "Member", permissions: [] };
    const user = userEvent.setup();
    renderMenu();
    await user.click(await screen.findByText("@onik97"));

    expect(screen.queryByRole("link", { name: "Settings" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Support" })).toBeInTheDocument();
  });

  it("hides Members for a viewer without members:write", async () => {
    const user = userEvent.setup();
    renderMenu();
    await user.click(await screen.findByText("@onik97"));

    expect(screen.queryByRole("link", { name: "Members" })).not.toBeInTheDocument();
  });

  it("shows Members for a viewer with members:write", async () => {
    roleResponse = { role_name: "Owner", permissions: ["members:write"] };
    const user = userEvent.setup();
    renderMenu();
    await user.click(await screen.findByText("@onik97"));

    expect(screen.getByRole("link", { name: "Members" })).toHaveAttribute("href", "/members");
  });

  it("calls logout and redirects to the home page when Logout is clicked", async () => {
    const logout = vi.fn();
    useSessionStore.setState({ logout });
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter initialEntries={["/docs/d-1"]}>
          <Routes>
            <Route path="/docs/:docId" element={<AccountMenu collapsed={false} />} />
            <Route path="/" element={<div>Home page</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await user.click(await screen.findByText("@onik97"));
    await user.click(screen.getByRole("button", { name: "Logout" }));

    expect(logout).toHaveBeenCalledOnce();
    expect(await screen.findByText("Home page")).toBeInTheDocument();
  });

  it("renders an icon-only trigger when collapsed", async () => {
    renderMenu(true);
    expect(screen.queryByText("@onik97")).not.toBeInTheDocument();
    expect(
      await screen.findByRole("button", { name: "Account menu for @onik97" }),
    ).toBeInTheDocument();
  });

  it("renders the avatar in the sidebar's shared size-8 identity slot", async () => {
    renderMenu();
    const avatar = await screen.findByText("O");
    expect(avatar).toHaveClass("size-8");
  });

  it("renders the GitHub avatar image when the user has one, not initials", async () => {
    ownerUser.avatar_url = "https://avatars.example/onik.png";
    try {
      renderMenu();
      const trigger = await screen.findByText("@onik97");
      const img = trigger.closest("button")?.querySelector("img");
      expect(img).toHaveAttribute("src", "https://avatars.example/onik.png");
      expect(screen.queryByText("O")).not.toBeInTheDocument();
    } finally {
      ownerUser.avatar_url = "";
    }
  });

  it("shows the resolved role name for the currently selected workspace as plain text under the login", async () => {
    renderMenu();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));

    const role = await screen.findByText("Owner");
    expect(role.closest("button")).toBe(screen.getByText("@onik97").closest("button"));
    expect(role.closest("[data-slot='no-fill-badge']")).toBeNull();
  });

  it("shows a custom role's real name instead of a hardcoded string", async () => {
    roleResponse = { role_name: "Editor" };
    renderMenu();
    expect(await screen.findByText("Editor")).toBeInTheDocument();
  });

  it("re-fetches the role when the selected workspace changes", async () => {
    renderMenu();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));

    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-2" });
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-2/me"));
  });
});
