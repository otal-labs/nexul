import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DeploySidebarNav } from "@/components/sidebar/DeploySidebarNav";
import { useSidebarStore } from "@/stores/sidebarStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

const renderNav = async (collapsed = false) => {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <DeploySidebarNav collapsed={collapsed} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await screen.findByRole("link", { name: /Configuration/ });
};

beforeEach(() => {
  useSidebarStore.setState({ workspaceNavOpen: true });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: { user: {}, instance_permissions: [] } };
    return { data: { role_name: "Owner", permissions: ["automations:read", "runners:read", "topology:read", "roles:write"] } };
  });
});

describe("DeploySidebarNav", () => {
  it("folds down to its header and opens again", async () => {
    const user = userEvent.setup();
    await renderNav();

    const header = screen.getByRole("button", { name: "Workspace" });
    expect(screen.getByRole("link", { name: /Runners/ })).toHaveAttribute("href", "/runners");

    await user.click(header);
    expect(header).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("link", { name: /Runners/ })).not.toBeInTheDocument();
    expect(useSidebarStore.getState().workspaceNavOpen).toBe(false);

    await user.click(header);
    expect(screen.getByRole("link", { name: /Runners/ })).toBeInTheDocument();
  });

  it("lists Configuration after Automations", async () => {
    await renderNav();
    const links = screen.getAllByRole("link").map((link) => link.getAttribute("href"));
    expect(links).toEqual(["/runners", "/topology", "/automations", "/configuration"]);
  });

  it("leaves Configuration out for a viewer holding instance permissions but no workspace section", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: {}, instance_permissions: ["instance:read", "connectors:read", "accounts:read"] } };
      return { data: { role_name: "Member", permissions: ["runners:read"] } };
    });
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <DeploySidebarNav collapsed={false} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await screen.findByRole("link", { name: /Runners/ });
    expect(screen.queryByRole("link", { name: /Configuration/ })).not.toBeInTheDocument();
  });

  it("tags only Automations as work in progress", async () => {
    await renderNav();
    expect(screen.getByRole("link", { name: /Automations/ })).toHaveTextContent("WIP");
    expect(screen.getByRole("link", { name: /Topology/ })).not.toHaveTextContent("WIP");
  });

  it("collapsed rail always shows the icons, with no header to fold", async () => {
    useSidebarStore.setState({ workspaceNavOpen: false });
    await renderNav(true);

    expect(screen.queryByRole("button", { name: "Workspace" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Runners" })).toHaveAttribute("title", "Runners");
  });
});
