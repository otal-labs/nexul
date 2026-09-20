import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SettingsPage } from "@/pages/SettingsPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, patch: mocks.patch, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const settings = {
  instance_url: "https://deploy.example.com",
  settings_version: 2,
  oauth_callback: "https://deploy.example.com/auth/callback",
  mention_chip_template: "{ticket.Ticket} {ticket.Status}",
};

const patList = (tokens: unknown[]) => ({ tokens });

// The page issues several GETs (settings, token list, connectors); route by URL so each resolves with the right shape.
const mockGet = (url: string) => {
  if (url === "/api/auth/tokens") {
    return Promise.resolve({ data: patList([]) });
  }
  if (url === "/api/connectors") {
    return Promise.resolve({ data: [] });
  }
  return Promise.resolve({ data: settings });
};

// Sections render one at a time off ?section=, so each test opens the page on the section it exercises.
const renderPage = (route = "/settings") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <SettingsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("SettingsPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
    mocks.get.mockImplementation(mockGet);
  });

  it("shows the current instance url and derived callback", async () => {
    renderPage();
    expect(await screen.findByDisplayValue("https://deploy.example.com")).toBeInTheDocument();
    expect(screen.getByText(/https:\/\/deploy\.example\.com\/auth\/callback/i)).toBeInTheDocument();
  });

  it("updates the instance url", async () => {
    mocks.put.mockResolvedValue({ data: { ...settings, instance_url: "https://new.example.com", settings_version: 3 } });
    const user = userEvent.setup();
    renderPage();

    const input = await screen.findByLabelText(/instance url/i);
    await user.clear(input);
    await user.type(input, "https://new.example.com");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).toHaveBeenCalledWith("/api/auth/settings", {
      instance_url: "https://new.example.com",
    });
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
    renderPage("/settings?section=tokens");

    await user.click(await screen.findByRole("button", { name: /generate connection token/i }));
    expect(await screen.findByText("header.payload.sig")).toBeInTheDocument();
  });

  it("mints a personal access token and shows it once", async () => {
    mocks.post.mockResolvedValue({
      data: {
        token: "dep_ABC123rawvalue",
        id: "pat-1",
        name: "ci agent",
        prefix: "rawvalue",
        created_at: "2026-08-12T00:00:00Z",
      },
    });
    const user = userEvent.setup();
    renderPage("/settings?section=tokens");

    await user.type(await screen.findByLabelText(/token name/i), "ci agent");
    await user.click(screen.getByRole("button", { name: /^create token$/i }));

    expect(await screen.findByText(/copy this token now/i)).toBeInTheDocument();
    expect(screen.getByText("dep_ABC123rawvalue")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/tokens", { name: "ci agent" });
  });

  it("lists personal access tokens with revoke state", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/auth/tokens") {
        return Promise.resolve({
          data: patList([
            {
              id: "pat-1",
              user_id: "u1",
              name: "ci agent",
              prefix: "abc123",
              created_at: "2026-08-01T00:00:00Z",
            },
            {
              id: "pat-2",
              user_id: "u1",
              name: "old token",
              prefix: "xyz789",
              created_at: "2026-07-01T00:00:00Z",
              revoked_at: "2026-07-15T00:00:00Z",
            },
          ]),
        });
      }
      if (url === "/api/connectors") return Promise.resolve({ data: [] });
      return Promise.resolve({ data: settings });
    });
    renderPage("/settings?section=tokens");

    expect(await screen.findByText("ci agent")).toBeInTheDocument();
    expect(screen.getByText("old token")).toBeInTheDocument();
    expect(screen.getByText(/revoked/)).toBeInTheDocument();

    const revokeButtons = screen.getAllByRole("button", { name: /^revoke$/i });
    expect(revokeButtons).toHaveLength(2);
    expect(revokeButtons.filter((b) => (b as HTMLButtonElement).disabled)).toHaveLength(1);
    expect(screen.getByText("old token").closest("li")).toBeInTheDocument();
  });

  it("revokes a personal access token", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/auth/tokens") {
        return Promise.resolve({
          data: patList([
            {
              id: "pat-1",
              user_id: "u1",
              name: "ci agent",
              prefix: "abc123",
              created_at: "2026-08-01T00:00:00Z",
            },
          ]),
        });
      }
      if (url === "/api/connectors") return Promise.resolve({ data: [] });
      return Promise.resolve({ data: settings });
    });
    mocks.del.mockResolvedValue({ data: patList([]) });
    const user = userEvent.setup();
    renderPage("/settings?section=tokens");

    const revoke = await screen.findByRole("button", { name: /^revoke$/i });
    await user.click(revoke);
    await user.click(screen.getByRole("button", { name: /^confirm$/i }));

    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/auth/tokens/pat-1"));
  });

  it("shows an error when settings fail to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Settings failed");
    renderPage();
    expect(await screen.findByText("Settings failed")).toBeInTheDocument();
  });
});

describe("SettingsPage mention chip layout gating", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("hides the mention chip layout panel without workspaces:write", async () => {
    mocks.get.mockImplementation(mockGet);
    renderPage("/settings?section=mentions");

    await screen.findByRole("heading", { name: "Settings" });
    expect(screen.queryByText("Mention chip layout")).not.toBeInTheDocument();
  });

  it("shows and lets a workspaces:write holder edit the template", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/workspaces/ws-1/me") {
        return Promise.resolve({ data: { role_name: "Owner", permissions: ["workspaces:write"] } });
      }
      return mockGet(url);
    });
    mocks.patch.mockResolvedValue({ data: { mention_chip_template: "{ticket.Status}" } });
    const user = userEvent.setup();
    renderPage("/settings?section=mentions");

    expect(await screen.findByText("Mention chip layout")).toBeInTheDocument();
    const section = within(screen.getByRole("region", { name: "Mention chip layout" }));
    const input = section.getByLabelText("Format");
    expect(input).toHaveValue("{ticket.Ticket} {ticket.Status}");

    await user.clear(input);
    await user.type(input, "{{ticket.Status}");
    await user.click(section.getByRole("button", { name: /^save$/i }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/auth/settings/mention-chip-template", {
      mention_chip_template: "{ticket.Status}",
    });
  });
});
