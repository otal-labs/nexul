import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PairingProjectsSection } from "@/components/settings/PairingProjectsSection";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), del: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const computer = (id: string, name: string) => ({
  id,
  name,
  server_url: `https://${name}.example.com`,
  token_expires_at: "2099-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "0.0.34",
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  position: 0,
  runs_at_once: 1,
  running: 0,
  chat_project_id: "",
  facts: { t3: { state: "answering" }, providers: [], projects: [{ id: "t3-app", title: "app", path: "/src/app", repository: "github.com/acme/web" }] },
});

const workspace = (id: string, name: string) => ({ id, name, slug: id, mention_chip_template: "", created_at: "", updated_at: "" });
const project = (id: string, name: string) => ({ id, name, prefix: name.slice(0, 3).toUpperCase(), position: 0 });

interface Fixture {
  computers: unknown[];
  workspaces: unknown[];
  projects: Record<string, unknown[]>;
  links: unknown[];
  presence?: Record<string, string>;
}

const SONNET = { data: { providers: [{ id: "claude", driver: "claudeAgent", name: "Claude", models: [{ slug: "sonnet-5", name: "Sonnet 5" }] }] } };

const serve = ({ computers, workspaces, projects, links, presence = { c1: "connected" } }: Fixture) =>
  mocks.get.mockImplementation(async (url: string, config?: { params?: { workspace_id?: string } }) => {
    if (url === "/api/pairing/computers") return { data: { computers } };
    if (url === "/api/pairing/presence") return { data: { computers: presence } };
    if (url === "/api/workspaces") return { data: workspaces };
    if (url === "/api/projects") return { data: projects[config?.params?.workspace_id ?? ""] ?? [] };
    if (url === "/api/pairing/projects") return { data: { links } };
    if (url === "/api/pairing/defaults") return { data: { placement: "in_order", clone_automatically: false } };
    if (url.startsWith("/api/projects/") && url.endsWith("/repos")) return { data: [{ owner: "acme", name: "web", full_name: "acme/web", connector_id: "github", role: "app" }] };
    if (url === "/api/pairing/computers/c1/projects") return { data: { projects: [{ id: "t3-app", title: "app", path: "/src/app" }] } };
    if (url.endsWith("/providers")) {
      if (presence[url.split("/")[4] ?? ""] !== "connected") throw new Error("the computer is offline");
      return SONNET;
    }
    throw new Error(`unexpected GET ${url}`);
  });

const renderTab = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={["/settings/pairing/projects"]}>
      <QueryClientProvider client={client}>
        <Routes>
          <Route path="/settings/:section/:tab?" element={<PairingProjectsSection />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
};

const oneWorkspace: Fixture = {
  computers: [computer("c1", "home")],
  workspaces: [workspace("w1", "Acme")],
  projects: { w1: [project("p-web", "Web"), project("p-api", "Api")] },
  links: [{ project_id: "p-web", computers: ["c1"], checkouts: [], provider: "claude", model: "sonnet-5", start_in: "worktree" }],
};

const row = (name: string) => screen.getByRole("button", { name: new RegExp(`^${name}`) }).closest("li") as HTMLElement;

describe("PairingProjectsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.del.mockReset();
    vi.mocked(toast.success).mockClear();
  });

  it("leads with Defaults and reads each project's own computers and model back by name, the rest as using defaults", async () => {
    serve(oneWorkspace);
    renderTab();

    expect(await screen.findByText("home · Sonnet 5 · New worktree")).toBeInTheDocument();
    expect(within(row("Api")).getByText("Uses defaults")).toBeInTheDocument();
    const rows = within(screen.getByRole("list", { name: "Defaults and projects" })).getAllByRole("button", { expanded: false });
    expect(rows[0]).toHaveTextContent("DefaultsFirst in order · Computer default · Project folder");
  });

  it("names a model from the first computer that is online when the first in order is not", async () => {
    const links = [{ project_id: "p-web", computers: [], checkouts: [], provider: "claude", model: "sonnet-5", start_in: "worktree" }];
    serve({ ...oneWorkspace, computers: [computer("c1", "home"), computer("c2", "vps")], links, presence: { c2: "connected" } });
    renderTab();

    expect(await screen.findByText("Sonnet 5 · New worktree")).toBeInTheDocument();
  });

  it("lists every workspace's projects in one list, with no group per workspace", async () => {
    serve({ ...oneWorkspace, workspaces: [workspace("w1", "Acme"), workspace("w2", "Otal")], projects: { w1: [project("p-web", "Web")], w2: [project("p-site", "Site")] } });
    renderTab();

    const list = await screen.findByRole("list", { name: "Defaults and projects" });
    expect(await within(list).findByRole("button", { name: /^Site/ })).toBeInTheDocument();
    expect(within(list).getByRole("button", { name: /^Web/ })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Otal" })).not.toBeInTheDocument();
  });

  it("opens one row at a time and saves a project's own computers", async () => {
    serve(oneWorkspace);
    mocks.put.mockImplementation(async (_url: string, body: object) => ({ data: { project_id: "p-api", ...body } }));
    const user = userEvent.setup();
    renderTab();

    await user.click(await screen.findByRole("button", { name: /^Web/ }));
    expect(within(row("Web")).getByRole("button", { name: "Use defaults" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^Api/ }));
    expect(within(row("Web")).queryByRole("button", { name: "Save" }), "opening another project closes this one").not.toBeInTheDocument();

    const api = within(row("Api"));
    expect(api.queryByRole("button", { name: "Use defaults" }), "nothing to clear on a project that uses the defaults").not.toBeInTheDocument();
    await user.click(await api.findByRole("radio", { name: "Custom" }));
    await user.click(api.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/pairing/projects/p-api", expect.objectContaining({ computers: ["c1"] })));
    expect(toast.success).toHaveBeenCalledWith("Api saved");
  });

  it("goes back to the defaults by clearing the project's link", async () => {
    serve(oneWorkspace);
    mocks.del.mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderTab();

    await user.click(await screen.findByRole("button", { name: /^Web/ }));
    await user.click(within(row("Web")).getByRole("button", { name: "Use defaults" }));

    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/projects/p-web"));
    expect(toast.success).toHaveBeenCalledWith("Web uses your defaults");
  });

  it("saves the defaults from the first row", async () => {
    serve(oneWorkspace);
    mocks.put.mockResolvedValue({ data: { provider: "claude", model: "sonnet-5", placement: "spread", clone_automatically: false, start_in: "worktree" } });
    const user = userEvent.setup();
    renderTab();

    await user.click(await screen.findByRole("button", { name: /^Defaults/ }));
    const defaults = within(row("Defaults"));
    await pickOption(user, /send runs to/i, "Least busy");
    await user.click(await defaults.findByRole("combobox", { name: "Model" }));
    await user.click(await screen.findByRole("option", { name: "Sonnet 5" }));
    await pickOption(user, /threads start in/i, "New worktree per thread");
    await user.click(defaults.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/api/pairing/defaults", {
        provider: "claude",
        model: "sonnet-5",
        model_options: [],
        start_in: "worktree",
        placement: "spread",
        clone_automatically: false,
      }),
    );
    expect(await defaults.findByRole("button", { name: "Saved" })).toBeInTheDocument();
  });

  it("points to the Computers tab while no computer is paired", async () => {
    serve({ ...oneWorkspace, computers: [] });
    renderTab();

    expect(await screen.findByRole("link", { name: "Computers" })).toHaveAttribute("href", "/settings/pairing");
    expect(screen.queryByText("Web")).not.toBeInTheDocument();
  });
});
