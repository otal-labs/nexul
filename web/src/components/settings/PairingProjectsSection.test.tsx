import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PairingProjectsSection } from "@/components/settings/PairingProjectsSection";

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
});

const workspace = (id: string, name: string) => ({ id, name, slug: id, mention_chip_template: "", created_at: "", updated_at: "" });
const project = (id: string, name: string) => ({ id, name, prefix: name.slice(0, 3).toUpperCase(), position: 0 });

interface Fixture {
  computers: unknown[];
  workspaces: unknown[];
  projects: Record<string, unknown[]>;
  links: unknown[];
}

const serve = ({ computers, workspaces, projects, links }: Fixture) =>
  mocks.get.mockImplementation(async (url: string, config?: { params?: { workspace_id?: string } }) => {
    if (url === "/api/pairing/computers") return { data: { computers } };
    if (url === "/api/workspaces") return { data: workspaces };
    if (url === "/api/projects") return { data: projects[config?.params?.workspace_id ?? ""] ?? [] };
    if (url === "/api/pairing/projects") return { data: { links } };
    if (url === "/api/pairing/computers/c1/projects") return { data: { projects: [{ id: "t3-app", title: "app", path: "/src/app" }] } };
    if (url === "/api/pairing/computers/c1/providers") {
      return { data: { providers: [{ id: "claude", driver: "claudeAgent", name: "Claude", models: [{ slug: "sonnet-5", name: "Sonnet 5" }] }] } };
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
  links: [{ project_id: "p-web", computer_id: "c1", harness_project_id: "t3-app", provider: "claude", model: "sonnet-5", start_in: "worktree" }],
};

const row = (name: string) => screen.getByText(name, { selector: "p" }).closest("li") as HTMLElement;

describe("PairingProjectsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.del.mockReset();
    vi.mocked(toast.success).mockClear();
  });

  it("reads each project's own link back by name, and the rest as using your defaults", async () => {
    serve(oneWorkspace);
    renderTab();

    expect(await screen.findByText("home · app · Sonnet 5 · New worktree")).toBeInTheDocument();
    expect(within(row("Api")).getByText("Uses your defaults")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Acme" }), "one workspace needs no group label").not.toBeInTheDocument();
  });

  it("groups projects under each workspace when there are several", async () => {
    serve({ ...oneWorkspace, workspaces: [workspace("w1", "Acme"), workspace("w2", "Otal")], projects: { w1: [project("p-web", "Web")], w2: [project("p-site", "Site")] } });
    renderTab();

    const otal = await screen.findByRole("region", { name: "Otal projects" });
    expect(within(otal).getByRole("heading", { name: "Otal" })).toBeInTheDocument();
    expect(await within(otal).findByText("Site")).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "Acme projects" })).getByText("Web")).toBeInTheDocument();
  });

  it("opens one project at a time and saves the link to that project", async () => {
    serve(oneWorkspace);
    mocks.put.mockResolvedValue({ data: { project_id: "p-api", computer_id: "c1", harness_project_id: "t3-app" } });
    const user = userEvent.setup();
    renderTab();

    await user.click(await screen.findByRole("button", { name: "Open Web's link" }));
    expect(within(row("Web")).getByRole("button", { name: "Use my defaults" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Open Api's link" }));
    expect(within(row("Web")).queryByRole("button", { name: "Save" }), "opening another project closes this one").not.toBeInTheDocument();

    const api = within(row("Api"));
    expect(api.queryByRole("button", { name: "Use my defaults" }), "nothing to clear on an unlinked project").not.toBeInTheDocument();
    await user.click(api.getByRole("combobox", { name: /computer/i }));
    await user.click(await screen.findByRole("option", { name: "home" }));
    await user.click(await api.findByRole("combobox", { name: /t3 project/i }));
    await user.click(await screen.findByRole("option", { name: /app/ }));
    await user.click(api.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/pairing/projects/p-api", expect.objectContaining({ computer_id: "c1", harness_project_id: "t3-app" })));
    expect(toast.success).toHaveBeenCalledWith("Api linked");
  });

  it("goes back to your defaults by clearing the project's link", async () => {
    serve(oneWorkspace);
    mocks.del.mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderTab();

    await user.click(await screen.findByRole("button", { name: "Open Web's link" }));
    await user.click(within(row("Web")).getByRole("button", { name: "Use my defaults" }));

    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/projects/p-web"));
    expect(toast.success).toHaveBeenCalledWith("Web uses your defaults");
  });

  it("points to the Computers tab while no computer is paired", async () => {
    serve({ ...oneWorkspace, computers: [] });
    renderTab();

    expect(await screen.findByRole("link", { name: "Computers" })).toHaveAttribute("href", "/settings/pairing");
    expect(screen.queryByText("Web")).not.toBeInTheDocument();
  });
});
