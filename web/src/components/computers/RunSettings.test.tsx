import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";
import { renderComputers } from "@/test/agentComputers";
import { pickOption } from "@/test/pickOption";
import { asOwner } from "@/test/viewer";

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

const project = (id: string, name: string) => ({ id, name, prefix: name.slice(0, 3).toUpperCase(), position: 0 });

interface Fixture {
  computers: unknown[];
  projects: unknown[];
  links: unknown[];
  presence?: Record<string, string>;
}

const SONNET = { providers: [{ id: "claude", driver: "claudeAgent", name: "Claude", models: [{ slug: "sonnet-5", name: "Sonnet 5" }] }] };

const serve = ({ computers, projects, links, presence = { c1: "connected" } }: Fixture) => {
  const rest = (url: string) => {
    if (url === "/api/pairing/computers") return { computers };
    if (url === "/api/pairing/presence") return { computers: presence };
    if (url === "/api/runners/enrollments") return [];
    if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme", slug: "acme" }];
    if (url === "/api/projects") return projects;
    if (url === "/api/pairing/projects") return { links };
    if (url === "/api/pairing/defaults") return { placement: "in_order", clone_automatically: false };
    if (url.startsWith("/api/projects/") && url.endsWith("/repos")) return [{ owner: "acme", name: "web", full_name: "acme/web", connector_id: "github", role: "app" }];
    if (url === "/api/pairing/computers/c1/projects") return { projects: [{ id: "t3-app", title: "app", path: "/src/app" }] };
    if (url.endsWith("/providers")) {
      if (presence[url.split("/")[4] ?? ""] !== "connected") throw new Error("the computer is offline");
      return SONNET;
    }
    throw new Error(`unexpected GET ${url}`);
  };
  mocks.get.mockImplementation(asOwner(rest));
};

const oneWorkspace: Fixture = {
  computers: [computer("c1", "home")],
  projects: [project("p-web", "Web"), project("p-api", "Api")],
  links: [{ project_id: "p-web", computers: ["c1"], checkouts: [], provider: "claude", model: "sonnet-5", start_in: "worktree" }],
};

const projectsGroup = () => screen.findByRole("region", { name: "Where runs go" });

describe("the projects on the Agent page", () => {
  beforeEach(() => {
    for (const fn of Object.values(mocks)) fn.mockReset();
    vi.mocked(toast.success).mockClear();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("leads with Defaults and reads each project's own computers and model back by name, the rest as using defaults", async () => {
    serve(oneWorkspace);
    renderComputers("/acme/agent?computers=");

    expect(await screen.findByText("home · Sonnet 5 · New worktree")).toBeInTheDocument();
    const group = await projectsGroup();
    expect(within(group).getByRole("link", { name: /Api/ })).toHaveTextContent("Uses defaults");
    const links = within(group).getAllByRole("link");
    expect(links[0]).toHaveTextContent("DefaultsFirst in your order · Computer default · Project folder");
  });

  it("names a model from the first computer that is online when the first in order is not", async () => {
    const links = [{ project_id: "p-web", computers: [], checkouts: [], provider: "claude", model: "sonnet-5", start_in: "worktree" }];
    serve({ ...oneWorkspace, computers: [computer("c1", "home"), computer("c2", "vps")], links, presence: { c2: "connected" } });
    renderComputers("/acme/agent?computers=");

    expect(await screen.findByText("Sonnet 5 · New worktree")).toBeInTheDocument();
  });

  it("saves a project's own computers, with nothing to clear on a project that uses the defaults", async () => {
    serve(oneWorkspace);
    mocks.put.mockImplementation(async (_url: string, body: object) => ({ data: { project_id: "p-api", ...body } }));
    const user = userEvent.setup();
    renderComputers("/acme/agent?computers=project%3Ap-web");

    expect(await screen.findByRole("heading", { name: "Web", level: 3 })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "Use defaults" })).toBeInTheDocument();

    await user.click(within(await projectsGroup()).getByRole("link", { name: /Api/ }));
    expect(await screen.findByRole("heading", { name: "Api", level: 3 })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Use defaults" }), "nothing to clear on a project that uses the defaults").not.toBeInTheDocument();
    await user.click(await screen.findByRole("radio", { name: "Only these" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/pairing/projects/p-api", expect.objectContaining({ computers: ["c1"] })));
    expect(toast.success).toHaveBeenCalledWith("Api saved");
  });

  it("goes back to the defaults by clearing the project's link", async () => {
    serve(oneWorkspace);
    mocks.del.mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderComputers("/acme/agent?computers=project%3Ap-web");

    await user.click(await screen.findByRole("button", { name: "Use defaults" }));

    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/projects/p-web"));
    expect(toast.success).toHaveBeenCalledWith("Web uses your defaults");
  });

  it("saves the defaults from the Defaults pane", async () => {
    serve(oneWorkspace);
    mocks.put.mockResolvedValue({ data: { provider: "claude", model: "sonnet-5", placement: "spread", clone_automatically: false, start_in: "worktree" } });
    const user = userEvent.setup();
    renderComputers("/acme/agent?computers=defaults");

    expect(await screen.findByRole("heading", { name: "Defaults", level: 3 })).toBeInTheDocument();
    await pickOption(user, /send runs to/i, "Least busy");
    await user.click(await screen.findByRole("combobox", { name: "Model" }));
    await user.click(await screen.findByRole("option", { name: "Sonnet 5" }));
    await pickOption(user, /threads start in/i, "New worktree per thread");
    await user.click(screen.getByRole("button", { name: "Save" }));

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
    expect(await screen.findByRole("button", { name: "Saved" })).toBeInTheDocument();
  });
});
