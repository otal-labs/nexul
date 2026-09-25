import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, useLocation } from "react-router";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ComputersSection } from "@/components/settings/ComputersSection";

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

vi.mock("@/components/settings/ComputerMCPToken", () => ({ ComputerMCPToken: () => null }));

const computerList = (computers: unknown[]) => ({ computers });

const computer = (overrides: Record<string, unknown> = {}) => ({
  id: "c1",
  name: "Home",
  server_url: "https://home.example.com",
  token_expires_at: "2099-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "0.0.34",
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const unconfirmed = { computer_id: "c1", confirmed_at: null, providers: [], turns: [] };
let setup: Record<string, unknown> = unconfirmed;

const serveComputers = (computers: unknown[]) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url.endsWith("/setup")) return { data: setup };
    return { data: computerList(computers) };
  });

const renderSection = (url = "/settings?section=pairing") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const location: { search: string } = { search: "" };
  const LocationProbe = () => {
    location.search = useLocation().search;
    return null;
  };
  const view = render(
    <MemoryRouter initialEntries={[url]}>
      <QueryClientProvider client={client}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ComputersSection />
        <LocationProbe />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { ...view, client, location };
};

describe("ComputersSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
    serveComputers([]);
  });

  it("shows an empty state when there are no computers", async () => {
    renderSection();
    expect(await screen.findByText(/no computers paired yet/i)).toBeInTheDocument();
  });

  it("lists paired computers", async () => {
    serveComputers([computer()]);
    renderSection();

    expect(await screen.findByText("Home")).toBeInTheDocument();
    expect(screen.getByText(/https:\/\/home\.example\.com.*0\.0\.34/)).toBeInTheDocument();
  });

  it("flags a computer expiring within the warning window", async () => {
    const soon = new Date(Date.now() + 2 * 24 * 60 * 60 * 1000).toISOString();
    serveComputers([computer({ token_expires_at: soon })]);
    renderSection();

    expect(await screen.findByText(/expires in \dd/)).toBeInTheDocument();
  });

  it("flags an expired computer as acting like unpaired", async () => {
    const past = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    serveComputers([computer({ token_expires_at: past })]);
    renderSection();

    expect(await screen.findByText(/acts as unpaired/i)).toBeInTheDocument();
  });

  it("reads a computer tunnel with no session yet as pairing in progress, not expired", async () => {
    serveComputers([
      computer({
        server_url: "https://laptop-ab12cd34.example.com",
        token_expires_at: "0001-01-01T00:00:00Z",
        harness_version: "",
        tunnel: { tunnel_id: "tun-1", hostname: "laptop-ab12cd34.example.com" },
      }),
    ]);
    renderSection();

    expect(await screen.findByText(/pairing in progress/i)).toBeInTheDocument();
    expect(screen.queryByText(/acts as unpaired/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^pair$/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /set up/i })).not.toBeInTheDocument();
  });

  it("badges an unconfirmed computer and opens the dialog straight at Set up for it", async () => {
    serveComputers([computer()]);
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByText("Needs setup")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^set up$/i }));
    expect(await screen.findByRole("dialog", { name: /set up home/i })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: /start setup/i })).toBeInTheDocument();
  });

  it("opens the named computer's Set up step from the setup link, and closing it forgets the link", async () => {
    serveComputers([computer(), computer({ id: "c2", name: "Mint" })]);
    const user = userEvent.setup();
    const { location } = renderSection("/settings?section=pairing&setup=c2");

    expect(await screen.findByRole("dialog", { name: /set up mint/i })).toBeInTheDocument();
    expect(screen.queryByRole("dialog", { name: /set up home/i })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(location.search).toBe("?section=pairing");
  });

  it("lists each provider with its confirmed-at time and follows a setup push without a refresh, changing nothing itself", async () => {
    const at = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
    setup = {
      computer_id: "c1",
      confirmed_at: at,
      providers: [{ provider: "codex", confirmed_at: at, skills: ["tdd"], skills_version: "v1", skills_outdated: false }],
      turns: [{ run_id: "r0", turn_id: "t0", provider: "codex", provider_name: "Codex", state: "confirmed", status: "Confirmed", updated_at: at }],
    };
    serveComputers([computer()]);
    const { client } = renderSection();

    expect(await screen.findByText("Setup confirmed")).toBeInTheDocument();
    expect(screen.getByText("Codex")).toBeInTheDocument();
    expect(screen.getByText("confirmed 2d ago")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /re-run setup/i })).toBeInTheDocument();

    setup = {
      ...setup,
      turns: [{ run_id: "r1", turn_id: "t1", provider: "codex", provider_name: "Codex", state: "running", status: "Installing", updated_at: at }],
    };
    await act(() => client.invalidateQueries({ queryKey: ["getComputerSetup"] }));
    expect(await screen.findByText("setting up…")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
    expect(mocks.put).not.toHaveBeenCalled();
    expect(mocks.del).not.toHaveBeenCalled();
  });

  it("signals out-of-date skills on a confirmed provider and points at re-running setup", async () => {
    const at = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
    setup = {
      computer_id: "c1",
      confirmed_at: at,
      providers: [{ provider: "codex", confirmed_at: at, skills: ["tdd"], skills_version: "old", skills_outdated: true }],
      turns: [],
    };
    serveComputers([computer()]);
    renderSection();

    expect(await screen.findByText("skills out of date")).toBeInTheDocument();
    expect(screen.getByText("Setup confirmed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /re-run setup/i })).toBeInTheDocument();
  });

  it("has no separate Pair by URL button; URL pairing lives in the dialog", async () => {
    renderSection();
    await screen.findByText(/no computers paired yet/i);
    expect(screen.queryByRole("button", { name: /pair by url/i })).not.toBeInTheDocument();
  });

  it("opens the pair-a-computer dialog on its Connect step", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /pair a computer/i }));
    expect(await screen.findByRole("dialog", { name: /pair a computer/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/computer name/i)).toBeInTheDocument();
  });

  it("re-pairs an existing computer, pre-filled with its name and URL", async () => {
    serveComputers([computer()]);
    mocks.post.mockResolvedValue({ data: computer({ harness_version: "0.0.35" }) });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /re-pair/i }));
    const nameInput = screen.getByLabelText(/^name$/i) as HTMLInputElement;
    const urlInput = screen.getByLabelText(/t3 server url/i) as HTMLInputElement;
    expect(nameInput.value).toBe("Home");
    expect(urlInput.value).toBe("https://home.example.com");

    await user.type(screen.getByLabelText(/one-time pairing token/i), "fresh-tok");
    await user.click(screen.getByRole("button", { name: /^re-pair$/i }));

    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/repair", {
        name: "Home",
        server_url: "https://home.example.com",
        token: "fresh-tok",
      }),
    );
    await waitFor(() =>
      expect(toast.success).toHaveBeenLastCalledWith(
        "Computer re-paired",
        expect.objectContaining({ description: expect.stringMatching(/previous session on this computer stays valid/) }),
      ),
    );
  });

  it("removes a computer after arming the confirm step", async () => {
    serveComputers([computer()]);
    mocks.del.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^remove$/i }));
    expect(mocks.del).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /^confirm$/i }));
    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/computers/c1"));
    await waitFor(() =>
      expect(toast.success).toHaveBeenLastCalledWith(
        "Computer removed",
        expect.objectContaining({ description: expect.stringMatching(/session on this computer stays valid until .*t3 auth session revoke/) }),
      ),
    );
  });

  it("removes a computer still pairing without a leftover-session note", async () => {
    serveComputers([computer({ token_expires_at: "0001-01-01T00:00:00Z" })]);
    mocks.del.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^remove$/i }));
    await user.click(screen.getByRole("button", { name: /^confirm$/i }));
    await waitFor(() => expect(toast.success).toHaveBeenLastCalledWith("Computer removed", undefined));
  });
});
