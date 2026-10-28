import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DeploySidebarNav } from "@/components/sidebar/DeploySidebarNav";
import { getMeKey } from "@/hooks/AuthHooks";
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
  await screen.findByRole("link", { name: /Runners/ });
};

beforeEach(() => {
  useSidebarStore.setState({ workspaceNavOpen: true });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: { user: {}, instance_permissions: [] } };
    if (url === "/api/machines") return { data: [{ id: "m-1", name: "box" }] };
    return { data: { role_name: "Owner", permissions: ["automations:read", "topology:read", "workspaces:write"] } };
  });
});

describe("DeploySidebarNav", () => {
  it("folds down to its header and opens again", async () => {
    const user = userEvent.setup();
    await renderNav();

    const header = screen.getByRole("button", { name: "Workspace" });
    expect(screen.getByRole("link", { name: /Runners/ })).toHaveAttribute("href", "/acme/runners");

    await user.click(header);
    expect(header).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("link", { name: /Runners/ })).not.toBeInTheDocument();
    expect(useSidebarStore.getState().workspaceNavOpen).toBe(false);

    await user.click(header);
    expect(screen.getByRole("link", { name: /Runners/ })).toBeInTheDocument();
  });

  it("leaves Configuration out for a viewer holding instance permissions but no workspace section", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: {}, instance_permissions: ["instance:read", "connectors:read", "accounts:read"] } };
      if (url === "/api/machines") return { data: [{ id: "m-1", name: "box" }] };
      return { data: { role_name: "Member", permissions: [] } };
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

  it.each([
    { who: "a Client with no machine and no way to add one", machines: [], me: { instance_permissions: [] }, link: false, readsMachines: true },
    { who: "someone who sees a machine", machines: [{ id: "m-1", name: "box" }], me: { instance_permissions: [] }, link: true, readsMachines: true },
    { who: "someone who can add machines", machines: [], me: { instance_permissions: ["machines:create"] }, link: true, readsMachines: false },
    { who: "the instance Owner", machines: [], me: { instance_permissions: [], instance_owner: true }, link: true, readsMachines: false },
  ])("shows the Runners link for $who: $link", async ({ machines, me, link, readsMachines }) => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: {}, ...me } };
      if (url === "/api/machines") return { data: machines };
      return { data: { role_name: "Member", permissions: ["topology:read"] } };
    });
    const client = new QueryClient();
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <DeploySidebarNav collapsed={false} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await screen.findByRole("link", { name: /Topology/ });
    await vi.waitFor(() => expect(client.getQueryState([getMeKey])?.status).toBe("success"));
    if (readsMachines) await vi.waitFor(() => expect(client.getQueryState(["machines"])?.status).toBe("success"));
    await vi.waitFor(() => expect(screen.queryByRole("link", { name: /Runners/ }) !== null).toBe(link));
    expect(vi.mocked(api.get).mock.calls.some(([url]) => url === "/api/machines")).toBe(readsMachines);
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
    expect(screen.getByRole("link", { name: "Runners" })).toBeInTheDocument();
    expect(screen.queryByText("Runners")).not.toBeInTheDocument();
  });
});
