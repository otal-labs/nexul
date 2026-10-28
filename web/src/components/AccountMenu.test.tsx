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
  first_login_done: true,
  created_at: "2026-08-12T12:00:00Z",
};

const held = (...names: string[]) => names.map((name) => ({ id: `r-${name}`, name, is_owner: name === "Owner" }));

const step = (key: string, state: "done" | "skipped" | "left", given = false) => ({ key, state, given });

let welcomeResponse: { first_run_done: boolean; steps: ReturnType<typeof step>[]; dot: boolean } = { first_run_done: true, steps: [], dot: false };

let roleResponse: { role_name: string; roles: ReturnType<typeof held>; permissions?: string[] } = {
  role_name: "Owner",
  roles: held("Owner"),
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
    roleResponse = { role_name: "Owner", roles: held("Owner"), permissions: ["projects:write"] };
    welcomeResponse = { first_run_done: true, steps: [], dot: false };
    // /api/auth/me returns the user; every other GET returns the workspace
    // role. Override the role via roleResponse, not by replacing this dispatcher.
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") {
        return { data: { user: ownerUser, needs_owner_wizard: false, needs_first_login_wizard: false } };
      }
      if (url === "/api/me/welcome") {
        return { data: welcomeResponse };
      }
      return { data: roleResponse };
    });
  });

  it("renders nothing while the profile hasn't loaded", () => {
    vi.mocked(api.get).mockImplementation(async () => new Promise(() => {}));
    const { container } = renderMenu();
    expect(container).toBeEmptyDOMElement();
  });

  it("opens the popover with Support and Sign out only", async () => {
    const user = userEvent.setup();
    renderMenu();
    await user.click(await screen.findByText("Onik"));

    expect(screen.queryByRole("link", { name: "Settings" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Members" })).not.toBeInTheDocument();
    const support = screen.getByRole("link", { name: "Support" });
    expect(support).toHaveAttribute("href", "https://github.com/otal-labs/nexul/issues");
    expect(support).toHaveAttribute("target", "_blank");
    expect(screen.getByRole("button", { name: "Sign out" })).toBeInTheDocument();
  });

  it("signs the session out server-side, clears local state, and redirects home when Sign out is clicked", async () => {
    const logout = vi.fn();
    useSessionStore.setState({ logout });
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
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
    await user.click(await screen.findByText("Onik"));
    await user.click(screen.getByRole("button", { name: "Sign out" }));

    expect(await screen.findByText("Home page")).toBeInTheDocument();
    expect(api.delete).toHaveBeenCalledWith("/api/auth/sessions/current");
    expect(logout).toHaveBeenCalledOnce();
  });

  it("still clears local state when the server-side sign-out fails", async () => {
    const logout = vi.fn();
    useSessionStore.setState({ logout });
    vi.mocked(api.delete).mockRejectedValue(new Error("offline"));
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
    await user.click(await screen.findByText("Onik"));
    await user.click(screen.getByRole("button", { name: "Sign out" }));

    expect(await screen.findByText("Home page")).toBeInTheDocument();
    expect(logout).toHaveBeenCalledOnce();
  });

  it("renders an icon-only trigger when collapsed", async () => {
    renderMenu(true);
    expect(
      await screen.findByRole("button", { name: "Account menu for Onik" }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Onik")).not.toBeInTheDocument();
  });

  it("shows the display name chosen in Your settings instead of the login", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") {
        const user = { ...ownerUser, display_name: "Bob Marsh" };
        return { data: { user, needs_owner_wizard: false, needs_first_login_wizard: false } };
      }
      if (url === "/api/me/welcome") {
        return { data: welcomeResponse };
      }
      return { data: roleResponse };
    });
    renderMenu();
    expect(await screen.findByText("Bob Marsh")).toBeInTheDocument();
    expect(screen.getByText("BM")).toBeInTheDocument();
    expect(screen.queryByText("@onik97")).not.toBeInTheDocument();
  });

  it("renders the GitHub avatar image when the user has one, not initials", async () => {
    ownerUser.avatar_url = "https://avatars.example/onik.png";
    try {
      renderMenu();
      const trigger = await screen.findByText("Onik");
      const img = trigger.closest("button")?.querySelector("img");
      expect(img).toHaveAttribute("src", "https://avatars.example/onik.png");
      expect(screen.queryByText("O")).not.toBeInTheDocument();
    } finally {
      ownerUser.avatar_url = "";
    }
  });

  it("shows the resolved role name for the currently selected workspace as plain text under the name", async () => {
    renderMenu();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));

    const role = await screen.findByText("Owner");
    expect(role.closest("button")).toBe(screen.getByText("Onik").closest("button"));
    expect(role.closest("[data-slot='no-fill-badge']")).toBeNull();
  });

  it("lists every role held, joined, and says when there is none", async () => {
    roleResponse = { role_name: "Developer, Frontend", roles: held("Developer", "Frontend") };
    const { unmount } = renderMenu();
    expect(await screen.findByText("Developer, Frontend")).toBeInTheDocument();
    unmount();

    roleResponse = { role_name: "", roles: [] };
    renderMenu();
    expect(await screen.findByText("No roles")).toBeInTheDocument();
  });

  it("re-fetches the role when the selected workspace changes", async () => {
    renderMenu();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));

    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-2" });
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-2/me"));
  });

  describe("welcome", () => {
    const dot = (button: HTMLElement) => button.querySelector(".bg-destructive");

    it("lights a dot on the trigger and names how many steps are left, in the sidebar and in the rail", async () => {
      welcomeResponse.steps = [step("github", "done"), step("computer", "left"), step("phone", "left")];
      const { unmount } = renderMenu();
      const trigger = (await screen.findByText("Onik")).closest("button")!;
      await waitFor(() => expect(dot(trigger)).not.toBeNull());
      expect(trigger).toHaveTextContent("2 welcome steps left");
      unmount();

      renderMenu(true);
      expect(await screen.findByRole("button", { name: "Account menu for Onik, 2 welcome steps left" })).toBeInTheDocument();
    });

    it.each([
      ["a skipped step", [step("github", "skipped")]],
      ["projects nobody gave", [step("projects", "left")]],
      ["every step done", [step("github", "done"), step("phone", "done")]],
    ])("keeps the dot off for %s", async (_, steps) => {
      welcomeResponse.steps = steps;
      renderMenu();
      const trigger = (await screen.findByText("Onik")).closest("button")!;
      await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/me/welcome"));
      await waitFor(() => expect(trigger).not.toHaveTextContent("welcome steps left"));
      expect(dot(trigger)).toBeNull();
    });

    it("tops the menu with Welcome and what is left, one row per step not done that opens the wizard on it", async () => {
      welcomeResponse.steps = [step("github", "done"), step("computer", "left"), step("projects", "left", true), step("phone", "skipped")];
      const user = userEvent.setup();
      renderMenu();
      await user.click(await screen.findByText("Onik"));

      expect(await screen.findByRole("link", { name: /Welcome/ })).toHaveAttribute("href", "/wizard/welcome");
      expect(screen.getByText("2 left")).toBeInTheDocument();
      expect(screen.getByRole("link", { name: "Your computer" })).toHaveAttribute("href", "/wizard/welcome?step=computer");
      expect(screen.getByRole("link", { name: "Your projects" })).toHaveAttribute("href", "/wizard/welcome?step=projects");
      expect(screen.getByRole("link", { name: /Get the phone app/ })).toHaveTextContent("Skipped");
      expect(screen.queryByRole("link", { name: "Connect GitHub" })).not.toBeInTheDocument();
    });

    it("keeps Welcome in the menu as done over total once every step is done", async () => {
      welcomeResponse.steps = [step("github", "done"), step("phone", "done")];
      const user = userEvent.setup();
      renderMenu();
      await user.click(await screen.findByText("Onik"));

      expect(await screen.findByText("2/2")).toBeInTheDocument();
      expect(screen.getAllByRole("link").map((link) => link.getAttribute("href"))).toEqual(["/wizard/welcome", "https://github.com/otal-labs/nexul/issues"]);
    });
  });
});
