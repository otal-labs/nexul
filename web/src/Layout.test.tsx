import { render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { Layout } from "@/Layout";
import { useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { buildLiveURL } from "@/lib/live";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/hooks/useLiveEvents", () => ({
  useLiveEvents: vi.fn(),
}));

vi.mock("@/hooks/NotificationHooks", () => ({
  useFetchNotifications: () => ({ data: [] }),
  useFetchUnreadCount: vi.fn(() => ({ data: { count: 0 } })),
  useMarkNotificationRead: () => ({ mutate: vi.fn(), isPending: false }),
  useMarkAllNotificationsRead: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api/client")>();
  return {
    ...actual,
    api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  };
});

// The server answers an Owner's `/me` with the whole grid; these are the actions the sidebar reads.
const ownerGrid = [
  "automations:read",
  "channels:delete",
  "channels:write",
  "chat:read",
  "chat:write",
  "docs:read",
  "docs:write",
  "members:write",
  "memories:read",
  "projects:read",
  "projects:write",
  "roles:write",
  "runners:read",
  "tickets:read",
  "topology:read",
];

// Sidebar entries are gated on the selected workspace's resolved `/me` permission list, so tests toggle this instead of a user field.
let meResponse = { role_name: "Owner", permissions: ownerGrid };
let projects: unknown[] = [];

const ownerUser = {
  id: "u1",
  provider: "github" as const,
  provider_user_id: "1",
  login: "onik97",
  name: "Onik",
  avatar_url: "",
  first_login_done: true,
  created_at: "2026-08-12T12:00:00Z",
};
let authUser = ownerUser;

const renderLayout = (path = "/") => {
  const client = new QueryClient();
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<div>page-content</div>} />
            <Route path="/wizard/onboarding/owner" element={<div>page-content</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("Layout", () => {
  beforeEach(() => {
    localStorage.clear();
    useSessionStore.setState({ token: null, isLoggedIn: false });
    meResponse = { role_name: "Owner", permissions: ownerGrid };
    projects = [];
    authUser = ownerUser;
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
    vi.mocked(api.get).mockReset();
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      // The auth profile (AccountMenu/WorkspaceSwitcher read it via useFetchMe) must match exactly before the workspace-scoped /me, which this URL also ends with.
      if (url === "/api/auth/me") {
        return { data: { user: authUser, needs_owner_wizard: false, needs_first_login_wizard: false, instance_permissions: meResponse.permissions } };
      }
      if (url.endsWith("/me")) return { data: meResponse };
      if (url === "/api/projects") return { data: projects };
      if (url === "/api/team") throw new Error("403");
      if (url === "/api/workspaces") return { data: [{ id: "ws-1", name: "Acme", slug: "acme", created_at: "", updated_at: "" }] };
      return { data: [] };
    });
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("renders the sidebar shell and outlet content for guests without nav links", () => {
    renderLayout();
    expect(screen.getByRole("link", { name: "Nexul" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /^Topology/ })).not.toBeInTheDocument();
    expect(screen.getByText("page-content")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /collapse sidebar|expand sidebar/i })).toBeInTheDocument();
  });

  it("shows the signed-in nav and account menu trigger", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(await screen.findByRole("link", { name: /^Topology/ })).toBeInTheDocument();
    expect(await screen.findByText("@onik97")).toBeInTheDocument();
  });

  it("has no Work/Deploy/Manage section header anywhere in the sidebar", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(screen.queryByText(/^Work$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Deploy$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Manage$/)).not.toBeInTheDocument();
  });

  it("offers New project in the sidebar when the workspace has no projects", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(await screen.findByRole("button", { name: "New project" })).toBeInTheDocument();
  });

  it("puts the collapse toggle outside the bottom control cluster, not next to the settings gear", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    const collapseButton = screen.getByRole("button", { name: /collapse sidebar|expand sidebar/i });
    const gear = screen.getByRole("link", { name: "Your settings" });
    expect(collapseButton.closest("div")).not.toBe(gear.closest("div"));
  });

  it("shows the Inbox link for signed-in users, above the project section", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    const inboxLink = screen.getByRole("link", { name: "Inbox" });
    expect(inboxLink).toHaveAttribute("href", "/acme/inbox");
    expect(inboxLink.compareDocumentPosition(await screen.findByRole("button", { name: "New project" }))).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    );
  });

  it("hides the Inbox link for guests", () => {
    renderLayout();
    expect(screen.queryByRole("link", { name: "Inbox" })).not.toBeInTheDocument();
  });

  it("shows the unread count badge on the Inbox link", () => {
    vi.mocked(useFetchUnreadCount).mockReturnValueOnce({ data: { count: 3 } } as ReturnType<
      typeof useFetchUnreadCount
    >);
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(screen.getByRole("link", { name: /Inbox/ })).toHaveTextContent("3");
  });

  it("reaches Configuration from the Workspace section and Your settings from the footer gear, with no Members or Settings links", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    await screen.findByText("@onik97");
    expect(await screen.findByRole("link", { name: "Configuration" })).toHaveAttribute("href", "/acme/configuration");
    expect(screen.getByRole("link", { name: "Your settings" })).toHaveAttribute("href", "/settings");
    expect(screen.queryByRole("link", { name: "Members" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Settings" })).not.toBeInTheDocument();
  });

  it("selects the first workspace when nothing is selected, even on an onboarding route with no switcher", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    useWorkspaceStore.setState({ selectedWorkspaceId: "" });
    renderLayout("/wizard/onboarding/owner");
    await vi.waitFor(() => expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-1"));
  });

  it("falls back to the first workspace when the persisted selection is no longer a membership", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-revoked" });
    renderLayout();
    await vi.waitFor(() => expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-1"));
  });

  it("hides the sidebar on onboarding routes so the wizard owns the viewport", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout("/wizard/onboarding/owner");
    expect(screen.getByText("page-content")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Nexul" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /collapse sidebar|expand sidebar/i })).not.toBeInTheDocument();
  });

  it("links to the runners page", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(await screen.findByRole("link", { name: /^Runners/ })).toBeInTheDocument();
  });

  const linkTargets = (nav: string) => {
    const region = screen.queryByRole("navigation", { name: nav });
    return region ? within(region).queryAllByRole("link").map((link) => link.getAttribute("href")) : [];
  };
  const createButtons = () =>
    ["New channel", "New voice channel", "New direct message"].filter((name) => screen.queryByRole("button", { name }));
  const member = ownerUser;

  it.each([
    {
      name: "the Owner sees every entry and every create action",
      user: ownerUser,
      permissions: ownerGrid,
      main: ["/acme/inbox", "/acme/chat", "/acme/board/BE", "/acme/projects/BE/interview", "/acme/docs", "/acme/memories", "/acme/projects/BE/settings"],
      workspace: ["/acme/runners", "/acme/topology", "/acme/automations", "/acme/configuration"],
      create: ["New channel", "New voice channel", "New direct message"],
    },
    {
      name: "a member who reads chat and tickets sees the board and nothing of the workspace",
      user: member,
      permissions: ["chat:read", "tickets:read"],
      main: ["/acme/inbox", "/acme/chat", "/acme/board/BE"],
      workspace: [],
      create: [],
    },
    {
      name: "a member who reads docs sees the one Docs entry and no Memories",
      user: member,
      permissions: ["docs:read"],
      main: ["/acme/inbox", "/acme/chat", "/acme/docs"],
      workspace: [],
      create: [],
    },
    {
      name: "a member who reads runners and topology sees only those two",
      user: member,
      permissions: ["runners:read", "topology:read"],
      main: ["/acme/inbox", "/acme/chat"],
      workspace: ["/acme/runners", "/acme/topology"],
      create: [],
    },
    {
      name: "a member who manages roles reaches Configuration alone",
      user: member,
      permissions: ["roles:write"],
      main: ["/acme/inbox", "/acme/chat"],
      workspace: ["/acme/configuration"],
      create: [],
    },
    {
      name: "a member who may only chat starts direct messages but not channels",
      user: member,
      permissions: ["chat:write"],
      main: ["/acme/inbox", "/acme/chat"],
      workspace: [],
      create: ["New direct message"],
    },
  ])("sidebar by permission: $name", async ({ user, permissions, main, workspace, create }) => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    authUser = user;
    meResponse = { role_name: "Member", permissions };
    projects = [{ id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" }];
    renderLayout();

    await waitFor(() => {
      expect(linkTargets("Main")).toEqual(main);
      expect(linkTargets("Workspace")).toEqual(workspace);
      expect(createButtons()).toEqual(create);
    });
    expect(screen.queryByRole("button", { name: "Workspace" }) !== null).toBe(workspace.length > 0);
  });
});

describe("buildLiveURL", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("appends the session token to a configured ws url", () => {
    vi.stubEnv("VITE_WS_URL", "ws://live:8080/ws/events");
    expect(buildLiveURL("tok/1")).toBe("ws://live:8080/ws/events?token=tok%2F1");
  });

  it("defaults to the API origin when VITE_WS_URL is unset", () => {
    vi.unstubAllEnvs();
    expect(buildLiveURL("tok")).toBe("ws://localhost:8080/ws/events?token=tok");
  });
});
