import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DnsOnboardingPage } from "@/pages/DnsOnboardingPage";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  errorMessage: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/wizard/onboarding/dns"]}>
        <Routes>
          <Route path="/wizard/onboarding/dns" element={<DnsOnboardingPage />} />
          <Route path="/" element={<div>home-page</div>} />
          <Route path="/settings" element={<div>settings-page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const zones = [{ id: "z1", name: "example.com", status: "active" }];

const cloudflareConnector = (configured: boolean) => [
  {
    connector: { id: "cloudflare", name: "Cloudflare", description: "", category: "infrastructure", icon: "cloudflare" },
    status: { configured },
    available: true,
    app_configured: false,
  },
];

const mockConnected = () =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/connectors") return { data: cloudflareConnector(true) };
    if (url === "/api/dns/zones") return { data: zones };
    return { data: [] };
  });

describe("DnsOnboardingPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.errorMessage.mockClear();
    mocks.toast.error.mockClear();
  });

  it("renders as the fourth wizard rung with a quiet skip link", async () => {
    mockConnected();
    renderPage();
    expect(await screen.findByRole("status", { name: /step 4 of 4/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /set up dns/i })).toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /skip for now/i }));
    expect(await screen.findByText("home-page")).toBeInTheDocument();
  });

  it("points at the connectors settings when Cloudflare is not connected", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/connectors") return { data: cloudflareConnector(false) };
      return { data: [] };
    });
    renderPage();
    expect(await screen.findByText(/connect cloudflare first/i)).toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /bare public address/i })).not.toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /go to settings/i }));
    expect(await screen.findByText("settings-page")).toBeInTheDocument();
  });

  it("offers the entry-path choice once Cloudflare is connected", async () => {
    mockConnected();
    renderPage();
    expect(await screen.findByRole("heading", { name: /how should traffic reach this instance/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /bare public address/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /cloudflare tunnel/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /reverse proxy/i })).toBeInTheDocument();
    expect(screen.queryByText(/gateways/i)).not.toBeInTheDocument();
  });

  it("walks the bare path through to Go live and on to the app", async () => {
    mockConnected();
    mocks.post.mockResolvedValue({ data: { id: "r1" } });
    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("radio", { name: /bare public address/i }));
    await user.click(screen.getByRole("button", { name: /^continue$/i }));
    await user.type(await screen.findByLabelText(/points to/i), "203.0.113.10");
    await user.click(screen.getByRole("button", { name: /create instance record/i }));

    expect(mocks.post).toHaveBeenCalledWith("/api/dns/instance-record", {
      zone_id: "z1",
      zone: "example.com",
      type: "A",
      target: "203.0.113.10",
    });
    await user.click(await screen.findByRole("button", { name: /continue to nexul/i }));
    expect(await screen.findByText("home-page")).toBeInTheDocument();
  });
});
