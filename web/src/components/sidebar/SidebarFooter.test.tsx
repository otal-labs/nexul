import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SidebarFooter } from "@/components/sidebar/SidebarFooter";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));
vi.mock("@/components/AccountMenu", () => ({ AccountMenu: () => <div data-testid="account-menu" /> }));
vi.mock("@/components/ThemeToggle", () => ({ ThemeToggle: () => <div data-testid="theme-toggle" /> }));

const renderFooter = (collapsed: boolean, isLoggedIn = true) =>
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <SidebarFooter collapsed={collapsed} isLoggedIn={isLoggedIn} />
      </QueryClientProvider>
    </MemoryRouter>,
  );

describe("SidebarFooter", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("shows the server version, muted and mono", async () => {
    mocks.get.mockResolvedValue({
      data: { version: "v0.2.0", channel: "stable", latest: null, update_available: false },
    });
    renderFooter(false);
    expect(await screen.findByText("v0.2.0")).toBeInTheDocument();
    expect(screen.queryByText(/Update available/)).not.toBeInTheDocument();
  });

  it("shows an update notice linking to the instance version settings section when one is available", async () => {
    mocks.get.mockResolvedValue({
      data: {
        version: "v0.2.0",
        channel: "stable",
        latest: { version: "v0.2.1", url: "https://github.com/otal-labs/nexul/releases/tag/v0.2.1" },
        update_available: true,
      },
    });
    renderFooter(false);
    const notice = await screen.findByText("Update available · v0.2.1");
    expect(notice).toHaveAttribute("href", "/settings?section=instance#instance-version");
  });

  it("hides the version line while the sidebar is collapsed", async () => {
    mocks.get.mockResolvedValue({
      data: { version: "v0.2.0", channel: "stable", latest: null, update_available: false },
    });
    renderFooter(true);
    await screen.findByTestId("theme-toggle");
    expect(screen.queryByText("v0.2.0")).not.toBeInTheDocument();
  });

  it("does not fetch the version when logged out", () => {
    renderFooter(false, false);
    expect(mocks.get).not.toHaveBeenCalled();
  });
});
