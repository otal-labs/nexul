import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SidebarFooter } from "@/components/sidebar/SidebarFooter";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));
vi.mock("@/components/AccountMenu", () => ({ AccountMenu: () => <div data-testid="account-menu" /> }));

const computer = { id: "c1", name: "Home", server_url: "https://home.example.com", token_expires_at: "2099-01-01T00:00:00Z", kind: "t3code" };

const serveSetup = (skillsOutdated: boolean) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url.endsWith("/setup"))
      return {
        data: {
          computer_id: "c1",
          confirmed_at: "2026-09-20T00:00:00Z",
          providers: [{ provider: "codex", confirmed_at: "2026-09-20T00:00:00Z", skills: [], skills_version: "v", skills_outdated: skillsOutdated }],
          turns: [],
        },
      };
    return { data: { computers: [computer] } };
  });

const renderFooter = () =>
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <SidebarFooter collapsed={false} />
      </QueryClientProvider>
    </MemoryRouter>,
  );

describe("SidebarFooter", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("shows the account menu beside a gear that opens Your settings", async () => {
    serveSetup(false);
    renderFooter();
    expect(screen.getByTestId("account-menu")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Your settings" })).toHaveAttribute("href", "/settings");
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/pairing/computers/c1/setup"));
    expect(screen.getByRole("link", { name: "Your settings" })).toBeInTheDocument();
  });

  it("marks the gear while one of your computers has out-of-date skills", async () => {
    serveSetup(true);
    renderFooter();
    expect(await screen.findByRole("link", { name: "Your settings, skills out of date" })).toHaveAttribute("href", "/settings");
  });
});
