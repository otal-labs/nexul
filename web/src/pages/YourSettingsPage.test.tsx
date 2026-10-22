import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
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

const me = {
  user: { id: "u1", login: "onik97", name: "Onik", avatar_url: "", display_name: "Onik N" },
};

const patList = (tokens: unknown[]) => ({ tokens });

const mockGet = (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: me });
  if (url === "/api/auth/tokens") return Promise.resolve({ data: patList([]) });
  if (url === "/api/auth/sessions") return Promise.resolve({ data: { sessions: [] } });
  if (url === "/api/auth/identities") return Promise.resolve({ data: { identities: [] } });
  if (url === "/api/auth/bootstrap-status") return Promise.resolve({ data: { configured: true } });
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
          <Route path="/settings/:section?/:tab?" element={<YourSettingsPage />} />
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
    expect(await screen.findByLabelText("Display name")).toHaveValue("Onik N");
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Security" })).toHaveAttribute("href", "/settings/security");
  });

  it("shows the gradient initials without a picture, and they follow the name as it is typed", async () => {
    const user = userEvent.setup();
    renderPage();
    const name = await screen.findByLabelText("Display name");
    expect(screen.getByText("ON")).toBeInTheDocument();

    await user.clear(name);
    await user.type(name, "Lena Kowalski");
    expect(screen.getByText("LK")).toBeInTheDocument();
  });

  it("marks Computers and its Computers tab while one of your computers has out-of-date skills", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/pairing/computers")
        return Promise.resolve({ data: { computers: [{ id: "c1", name: "Home", token_expires_at: "2099-01-01T00:00:00Z", kind: "t3code" }] } });
      if (url === "/api/pairing/computers/c1/setup")
        return Promise.resolve({
          data: { computer_id: "c1", confirmed_at: "2026-09-20T00:00:00Z", turns: [], providers: [{ provider: "codex", confirmed_at: "2026-09-20T00:00:00Z", skills: [], skills_version: "old", skills_outdated: true }] },
        });
      return mockGet(url);
    });
    renderPage("/settings/pairing");

    expect(await screen.findByRole("link", { name: "Computers, skills out of date" })).toHaveAttribute("href", "/settings/pairing");
    expect(screen.getByRole("tab", { name: "Computers, skills out of date" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Projects" })).toBeInTheDocument();
  });

  it("opens Security on the Devices tab with the desktop card and the device list", async () => {
    renderPage("/settings/security");
    expect(await screen.findByRole("tab", { name: "Devices", selected: true })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy connection token" })).toBeInTheDocument();
    expect(await screen.findByText(/no other devices signed in/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/token name/i)).not.toBeInTheDocument();
  });

  it("lands a computer setup link on the Computers tab with its setup param intact", async () => {
    renderPage("/settings/pairing?setup=c1");

    expect(await screen.findByRole("tab", { name: "Computers", selected: true })).toBeInTheDocument();
    expect(screen.getByText("Computers card")).toBeInTheDocument();
    expect(screen.getByLabelText("location")).toHaveTextContent("setup=c1");
  });

  it("opens the personal access tokens from their path", async () => {
    renderPage("/settings/security/tokens");

    expect(await screen.findByLabelText(/token name/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^create token$/i })).toBeInTheDocument();
  });
});
