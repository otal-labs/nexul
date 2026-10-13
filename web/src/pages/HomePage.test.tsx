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

const signInWith = (permissions: string[]) => {
  useSessionStore.setState({ isLoggedIn: true });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockImplementation(async (url: string) => ({
    data: url === "/api/workspaces" ? [{ id: "ws-1", name: "Acme", slug: "acme" }] : { role_name: "Member", roles: [{ id: "r-member", name: "Member", is_owner: false }], permissions },
  }));
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

  it.each([
    { permissions: ["tickets:read"], shown: "Open board", hidden: "Open topology" },
    { permissions: ["topology:read"], shown: "Open topology", hidden: "Open board" },
  ])("offers only the areas the member can read ($permissions)", async ({ permissions, shown, hidden }) => {
    signInWith(permissions);
    renderHome();

    expect(await screen.findByRole("link", { name: shown })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: hidden })).not.toBeInTheDocument();
  });

  it("keeps agent, trail, and play wording from a member who can neither run nor read plays", async () => {
    signInWith(["tickets:read", "docs:read"]);
    renderHome();

    expect(await screen.findByRole("link", { name: "Open board" })).toBeInTheDocument();
    expect(screen.queryByText(/agent/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/trail/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/play/i)).not.toBeInTheDocument();
  });

  it.each(["plays:run", "plays:read"])("shows the agent pitch to a member holding %s", async (permission) => {
    signInWith(["tickets:read", permission]);
    renderHome();

    expect(await screen.findByText(/each play run leaves a trail of every step/)).toBeInTheDocument();
    expect(screen.getByText("nexul · agent trail")).toBeInTheDocument();
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
