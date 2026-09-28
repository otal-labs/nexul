import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { YourSettingsPage } from "@/pages/YourSettingsPage";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: vi.fn(), patch: vi.fn(), delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/components/settings/ComputersSection", () => ({ ComputersSection: () => <p>Computers card</p> }));
vi.mock("@/components/settings/PairingDefaultsSection", () => ({ PairingDefaultsSection: () => <p>Defaults card</p> }));

const me = {
  user: { id: "u1", login: "onik97", name: "Onik", avatar_url: "", display_name: "Onik N", can_create_workspace: true },
};

const patList = (tokens: unknown[]) => ({ tokens });

const mockGet = (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: me });
  if (url === "/api/auth/tokens") return Promise.resolve({ data: patList([]) });
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

const LocationProbe = () => {
  const location = useLocation();
  return <output aria-label="location">{`${location.pathname}${location.search}${location.hash}`}</output>;
};

const renderPage = (route = "/settings") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/settings" element={<YourSettingsPage />} />
          <Route path="/configuration" element={<p>Configuration page</p>} />
        </Routes>
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("YourSettingsPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
    mocks.get.mockImplementation(mockGet);
  });

  it("opens on Profile with the saved display name in the form", async () => {
    renderPage();
    expect(screen.getByRole("heading", { name: "Settings" })).toBeInTheDocument();
    expect(await screen.findByLabelText("Name")).toHaveValue("Onik N");
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Security" })).toHaveAttribute("href", "/settings?section=security");
  });

  it.each([
    ["/settings?section=connectors&connector=github&connected=1", "/configuration?section=connectors&connector=github&connected=1"],
    ["/settings?section=instance#instance-version", "/configuration?section=instance#instance-version"],
    ["/settings?section=members", "/configuration?section=members"],
  ])("sends a moved section link %s to %s with its query and hash", async (from, to) => {
    renderPage(from);
    expect(await screen.findByText("Configuration page")).toBeInTheDocument();
    expect(screen.getByLabelText("location")).toHaveTextContent(to);
  });

  it("puts both token cards on the Security section with no tab row", async () => {
    renderPage("/settings?section=tokens&tab=personal");
    expect(await screen.findByRole("button", { name: /generate connection token/i })).toBeInTheDocument();
    expect(await screen.findByLabelText(/token name/i)).toBeInTheDocument();
    expect(screen.queryByRole("tablist")).not.toBeInTheDocument();
    expect(screen.getByLabelText("location")).toHaveTextContent("/settings?section=security&tab=tokens");
  });

  it("lands a computer setup link on the Computers tab with its setup param intact", async () => {
    const user = userEvent.setup();
    renderPage("/settings?section=pairing&setup=c1");

    expect(await screen.findByRole("tab", { name: "Computers", selected: true })).toBeInTheDocument();
    expect(screen.getByText("Computers card")).toBeInTheDocument();
    expect(screen.getByLabelText("location")).toHaveTextContent("setup=c1");

    await user.click(screen.getByRole("tab", { name: "Defaults" }));
    expect(screen.getByText("Defaults card")).toBeInTheDocument();
  });

  it("generates and reveals a connection token", async () => {
    mocks.post.mockResolvedValue({
      data: {
        token: "header.payload.sig",
        instance_url: "https://deploy.example.com",
        settings_version: 2,
        expires_at: "2026-09-11T12:00:00Z",
      },
    });
    const user = userEvent.setup();
    renderPage("/settings?section=security");

    await user.click(await screen.findByRole("button", { name: /generate connection token/i }));
    expect(await screen.findByText("header.payload.sig")).toBeInTheDocument();
  });

  it("mints a personal access token and shows it once", async () => {
    mocks.post.mockResolvedValue({
      data: { token: "dep_ABC123rawvalue", id: "pat-1", name: "ci agent", prefix: "rawvalue", created_at: "2026-08-12T00:00:00Z" },
    });
    const user = userEvent.setup();
    renderPage("/settings?section=security");

    await user.type(await screen.findByLabelText(/token name/i), "ci agent");
    await user.click(screen.getByRole("button", { name: /^create token$/i }));

    expect(await screen.findByText(/copy this token now/i)).toBeInTheDocument();
    expect(screen.getByText("dep_ABC123rawvalue")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/tokens", { name: "ci agent" });
  });

  it("lists personal access tokens with revoke state and revokes one", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/auth/tokens") {
        return Promise.resolve({
          data: patList([
            { id: "pat-1", user_id: "u1", name: "ci agent", prefix: "abc123", created_at: "2026-08-01T00:00:00Z" },
            { id: "pat-2", user_id: "u1", name: "old token", prefix: "xyz789", created_at: "2026-07-01T00:00:00Z", revoked_at: "2026-07-15T00:00:00Z" },
          ]),
        });
      }
      return mockGet(url);
    });
    mocks.del.mockResolvedValue({ data: patList([]) });
    const user = userEvent.setup();
    renderPage("/settings?section=security");

    expect(await screen.findByText("ci agent")).toBeInTheDocument();
    expect(screen.getByText("old token")).toBeInTheDocument();
    expect(screen.getByText(/revoked/)).toBeInTheDocument();

    const revokeButtons = screen.getAllByRole("button", { name: /^revoke$/i });
    expect(revokeButtons).toHaveLength(2);
    expect(revokeButtons.filter((b) => (b as HTMLButtonElement).disabled)).toHaveLength(1);

    await user.click(revokeButtons[0]!);
    await user.click(screen.getByRole("button", { name: /^confirm$/i }));
    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/auth/tokens/pat-1"));
  });
});
