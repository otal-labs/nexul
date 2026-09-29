import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConfigurationPage } from "@/pages/ConfigurationPage";
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
};

const workspaces = [{ id: "ws-1", name: "Acme", mention_chip_template: "{ticket.Ticket} {ticket.Status}", created_at: "", updated_at: "" }];

// The page issues several GETs (me, settings, connectors, version); route by URL so each resolves with the right shape.
const mockGet = (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: { can_create_workspace: true } } });
  if (url === "/api/connectors") return Promise.resolve({ data: [] });
  if (url === "/api/workspaces") return Promise.resolve({ data: workspaces });
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve({ data: { role_name: "Member", permissions: [] } });
  if (url.startsWith("/api/version")) return Promise.resolve({ data: { current: "0.1.0", channel: "beta" } });
  return Promise.resolve({ data: settings });
};

// Sections render one at a time off the path, so each test opens the page on the section it exercises.
const renderPage = (route = "/configuration") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/configuration/:section?" element={<ConfigurationPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("ConfigurationPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
    mocks.get.mockImplementation(mockGet);
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("shows the current instance url and derived callback", async () => {
    renderPage("/configuration/instance");
    expect(await screen.findByDisplayValue("https://deploy.example.com")).toBeInTheDocument();
    expect(screen.getByText(/https:\/\/deploy\.example\.com\/auth\/callback/i)).toBeInTheDocument();
  });

  it("updates the instance url", async () => {
    mocks.put.mockResolvedValue({ data: { ...settings, instance_url: "https://new.example.com", settings_version: 3 } });
    const user = userEvent.setup();
    renderPage("/configuration/instance");

    const input = await screen.findByLabelText(/instance url/i);
    await user.clear(input);
    await user.type(input, "https://new.example.com");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).toHaveBeenCalledWith("/api/auth/settings", {
      instance_url: "https://new.example.com",
    });
  });

  it("shows an error when settings fail to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Settings failed");
    renderPage();
    expect(await screen.findByText("Settings failed")).toBeInTheDocument();
  });
});

describe("ConfigurationPage mention chip layout gating", () => {
  beforeEach(() => {
    mocks.get.mockImplementation(mockGet);
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("hides the mention chip layout panel without workspaces:write", async () => {
    renderPage("/configuration/mentions");

    await screen.findByRole("heading", { name: "Configuration" });
    expect(screen.queryByText("Mention chip layout")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Mention chips" })).not.toBeInTheDocument();
  });

  it("shows and lets a workspaces:write holder edit the template", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/workspaces/ws-1/me") {
        return Promise.resolve({ data: { role_name: "Owner", permissions: ["workspaces:write"] } });
      }
      return mockGet(url);
    });
    mocks.patch.mockResolvedValue({ data: { ...workspaces[0], mention_chip_template: "{ticket.Status}" } });
    const user = userEvent.setup();
    renderPage("/configuration/mentions");

    expect(await screen.findByText("Mention chip layout")).toBeInTheDocument();
    const section = within(screen.getByRole("region", { name: "Mention chip layout" }));
    const input = section.getByLabelText("Format");
    expect(input).toHaveValue("{ticket.Ticket} {ticket.Status}");

    await user.clear(input);
    await user.type(input, "{{ticket.Status}");
    await user.click(section.getByRole("button", { name: /^save$/i }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/mention-chip-template", {
      mention_chip_template: "{ticket.Status}",
    });
  });
});
