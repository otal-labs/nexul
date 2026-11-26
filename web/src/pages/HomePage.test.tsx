import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { HomePage } from "@/pages/HomePage";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

const renderHome = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>
    </QueryClientProvider>,
  );

const statuses = [
  { id: "st-doing", name: "In progress", position: 0, kind: "progress", icon: "", created_at: "", updated_at: "" },
  { id: "st-done", name: "Done", position: 1, kind: "done", icon: "", created_at: "", updated_at: "" },
];
const ticket = (id: string, number: number, title: string, overrides: Record<string, unknown> = {}) => ({
  id, project_id: "p-1", category_id: "", type_id: "", title, body: "", status: "st-doing", position: 0, number, doc_id: "",
  developer: "sam", developer_id: "u-sam", tester: "", tester_id: "", reporter: { kind: "user" }, due_on: "",
  created_at: "", updated_at: "2026-11-20T00:00:00Z", labels: null, ...overrides,
});

const signInWith = (permissions: string[], data: Record<string, unknown> = {}) => {
  useSessionStore.setState({ isLoggedIn: true });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  const responses: Record<string, unknown> = {
    "/api/workspaces": [{ id: "ws-1", name: "Acme", slug: "acme" }],
    "/api/workspaces/ws-1/me": { role_name: "Member", roles: [{ id: "r-member", name: "Member", is_owner: false }], permissions },
    "/api/projects": [{ id: "p-1", name: "Atlas", prefix: "ATL", setup: { finished: true } }],
    "/api/auth/me": { user: { id: "u-sam", login: "sam" } },
    "/api/statuses": statuses,
    "/api/tickets": [],
    "/api/notifications": [],
    "/api/notifications/unread-count": { count: 0, workspaces: {} },
    "/api/plays/runs/active": { active: {}, waiting: {}, started: {} },
    ...data,
  };
  vi.mocked(api.get).mockImplementation(async (url: string) => ({ data: responses[url] ?? [] }));
};

beforeEach(() => {
  useSessionStore.setState({ isLoggedIn: false });
  vi.mocked(api.get).mockReset();
});

describe("HomePage", () => {
  it("renders the product pitch and a call to action", () => {
    renderHome();
    expect(screen.getByRole("heading", { name: "Docs, tickets, chat, and deploys in one place." })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Self-host your own" })).toHaveAttribute(
      "href",
      "https://nexul.io/docs/guide/install/",
    );
    expect(screen.getByRole("link", { name: "Self-host your own" })).toHaveAttribute("target", "_blank");
    expect(screen.getByRole("link", { name: "Self-host your own" })).toHaveAttribute("rel", "noreferrer");
  });

  it("lists the open tickets waiting on the viewer, not done ones or someone else's", async () => {
    signInWith(["tickets:read"], {
      "/api/tickets": [
        ticket("t-1", 12, "Fix the login redirect"),
        ticket("t-2", 13, "Shipped already", { status: "st-done" }),
        ticket("t-3", 14, "Lena's ticket", { developer: "lena", developer_id: "u-lena" }),
      ],
    });
    renderHome();

    expect(await screen.findByRole("link", { name: /Fix the login redirect/ })).toHaveAttribute("href", "/acme/tickets/ATL-12");
    expect(screen.queryByText("Shipped already")).not.toBeInTheDocument();
    expect(screen.queryByText("Lena's ticket")).not.toBeInTheDocument();
    expect(screen.getByText("ticket on you")).toBeInTheDocument();
  });

  it("says nothing waits on the viewer and the inbox is read when both are empty", async () => {
    signInWith(["tickets:read"]);
    renderHome();

    expect(await screen.findByText("Nothing is waiting on you in Atlas.")).toBeInTheDocument();
    expect(await screen.findByText("You're all caught up.")).toBeInTheDocument();
  });

  it("leaves the tickets and plays out for a member who reads neither", async () => {
    signInWith(["docs:read"]);
    renderHome();

    expect(await screen.findByText("You're all caught up.")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Your tickets" })).not.toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Plays running" })).not.toBeInTheDocument();
  });

  it("shows a run waiting for an answer to a member holding plays:read", async () => {
    signInWith(["tickets:read", "plays:read"], {
      "/api/tickets": [ticket("t-1", 12, "Fix the login redirect", { developer: "", developer_id: "" })],
      "/api/plays/runs/active": { active: { "t-1": "tr-1" }, waiting: { "t-1": "tr-1" }, started: { "t-1": "2026-11-20T00:00:00Z" } },
    });
    renderHome();

    const runs = await screen.findByRole("region", { name: "Plays running" });
    await vi.waitFor(() => expect(runs).toHaveTextContent(/Fix the login redirect.*ATL-12 · waiting for an answer/));
  });

  it("shows a member with no role why the workspace is empty and opens a DM with an Owner from it", async () => {
    useSessionStore.setState({ isLoggedIn: true });
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    const post = vi.fn().mockResolvedValue({ data: { id: "dm-1" } });
    Object.assign(api, { post });
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces") return { data: [{ id: "ws-1", name: "Acme", slug: "acme" }] };
      if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "", roles: [], permissions: [] } };
      if (url === "/api/workspaces/ws-1/people") return { data: { people: [{ user_id: "u-alice", login: "alice", display_name: "Alice", avatar_url: "", owner: true }, { user_id: "u-sam", login: "sam", display_name: "Sam", avatar_url: "" }] } };
      if (url === "/api/auth/me") return { data: { user: { id: "u-sam" } } };
      return { data: [] };
    });
    const user = userEvent.setup();
    renderHome();

    expect(await screen.findByRole("heading", { name: "No roles in Acme yet" })).toBeInTheDocument();
    expect(screen.getByText("Your roles decide what you see here, so there are no projects or plays for you yet. You can read public channels and message an Owner below, who can give you a role.")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open board" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Message Sam" })).not.toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: "Message Alice" }));

    await vi.waitFor(() => expect(post).toHaveBeenCalledWith("/api/chat/dms", { workspace_id: "ws-1", participant_ids: ["u-alice"] }));
  });
});
