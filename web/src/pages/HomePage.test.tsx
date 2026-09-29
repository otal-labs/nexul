import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
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
  vi.mocked(api.get).mockResolvedValue({ data: { role_name: "Member", permissions } });
};

beforeEach(() => {
  useSessionStore.setState({ isLoggedIn: false });
  vi.mocked(api.get).mockReset();
});

describe("HomePage", () => {
  it("renders the product pitch and a call to action", () => {
    renderHome();
    expect(
      screen.getByRole("heading", { name: "One button. The trail shows every step the agent took." }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Self-host your own" })).toHaveAttribute(
      "href",
      "https://nexul.io/docs/guide/install/",
    );
    expect(screen.getByRole("link", { name: "Self-host your own" })).toHaveAttribute("target", "_blank");
    expect(screen.getByRole("link", { name: "Self-host your own" })).toHaveAttribute("rel", "noreferrer");
  });

  it.each([
    { permissions: ["tickets:read"], shown: "Open the board", hidden: "View topology" },
    { permissions: ["topology:read"], shown: "View topology", hidden: "Open the board" },
  ])("offers only the areas the member can read ($permissions)", async ({ permissions, shown, hidden }) => {
    signInWith(permissions);
    renderHome();

    expect(await screen.findByRole("link", { name: shown })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: hidden })).not.toBeInTheDocument();
  });
});
