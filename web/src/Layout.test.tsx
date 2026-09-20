import { render, screen } from "@testing-library/react";
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

// The owner-only nav items (ticket 15) are gated on the selected workspace's resolved `/me` permission list, so tests toggle this instead of a user field to simulate a permission-holding vs. non-holding member.
let meResponse = { role_name: "Owner", permissions: ["projects:write", "members:write"] };

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
};const renderLayout = (path = "/") => {
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
    meResponse = { role_name: "Owner", permissions: ["projects:write", "members:write"] };
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
    vi.mocked(api.get).mockReset();
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      // The auth profile (AccountMenu/WorkspaceSwitcher read it via useFetchMe) must match exactly before the workspace-scoped /me, which this URL also ends with.
      if (url === "/api/auth/me") {
        return { data: { user: ownerUser, needs_owner_wizard: false, needs_first_login_wizard: false } };
      }
      if (url.endsWith("/me")) return { data: meResponse };
      if (url === "/api/workspaces") return { data: [{ id: "ws-1", name: "Acme", created_at: "", updated_at: "" }] };
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
    expect(screen.getByRole("link", { name: /^Topology/ })).toBeInTheDocument();
    expect(await screen.findByText("@onik97")).toBeInTheDocument();
  });

  it("has no leftover flat Docs nav link — docs live inside each project's tree row now", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(screen.queryByRole("link", { name: "Docs" })).not.toBeInTheDocument();
  });

  it("has no Work/Deploy/Manage section header anywhere in the sidebar", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(screen.queryByText(/^Work$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Deploy$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Manage$/)).not.toBeInTheDocument();
  });

  it("renders the project tree's persistent New project row in the sidebar", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(screen.getByRole("button", { name: "New project" })).toBeInTheDocument();
  });

  it("puts the collapse toggle outside the bottom control cluster, not next to the theme toggle", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    const collapseButton = screen.getByRole("button", { name: /collapse sidebar|expand sidebar/i });
    const themeToggle = screen.getByRole("button", { name: /switch to .* theme/i });
    expect(collapseButton.closest("div")).not.toBe(themeToggle.closest("div"));
  });

  it("shows the Inbox link for signed-in users, above the project tree", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    const inboxLink = screen.getByRole("link", { name: "Inbox" });
    expect(inboxLink).toHaveAttribute("href", "/inbox");
    expect(inboxLink.compareDocumentPosition(screen.getByRole("button", { name: "New project" }))).toBe(
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

  it("no longer renders Members as a sidebar nav link — it lives in the account menu", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    await screen.findByText("@onik97");
    expect(screen.queryByRole("link", { name: "Members" })).not.toBeInTheDocument();
  });

  it("no longer renders Settings as a sidebar nav link — it moved into the account menu", async () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    await screen.findByText("@onik97");
    expect(screen.queryByRole("link", { name: "Settings" })).not.toBeInTheDocument();
  });

  it("includes a theme toggle in the header", () => {
    renderLayout();
    expect(screen.getByRole("button", { name: /switch to .* theme/i })).toBeInTheDocument();
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

  it("links to the runners page", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    renderLayout();
    expect(screen.getByRole("link", { name: /^Runners/ })).toBeInTheDocument();
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
