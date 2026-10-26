import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { StackPage } from "@/pages/StackPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const access = vi.hoisted(() => ({ areas: ["projects", "topology", "editProjects", "editStacks", "editDeploys", "dns"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({
  useAreaAccess: () => (area: string) => access.areas.includes(area),
  useHasInstancePermission: (value: string) => access.areas.includes(value.split(":")[0] ?? ""),
}));

beforeEach(() => {
  access.areas = ["projects", "topology", "editProjects", "editStacks", "editDeploys", "dns"];
});

const manage = vi.hoisted(() => ({ stack: false }));
vi.mock("@/hooks/useCanManageAccess", () => ({ useCanManageAccess: (entity?: { type: string }) => entity?.type === "stack" && manage.stack }));

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), _delete: vi.fn() }));

// The viewer is an Owner with no rule on any item; every other read is the test's own.
vi.mock("@/api/client", async () => {
  const { withOwner } = await import("@/test/viewer");
  return { api: { get: withOwner(mocks.get), post: mocks.post, patch: mocks.patch, delete: mocks._delete }, errorMessage: () => "boom" };
});

const stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "instance",
  strategy: "compose",
  managed: true,
  created_at: "2026-08-12T09:00:00Z",
  updated_at: "2026-08-12T09:00:00Z",
};

const container = {
  id: "svc-1",
  stack_id: "stack-1",
  name: "api",
  declared: { image: "ghcr.io/onik/api:v1" },
  container_name: "api-api-1",
  image: "ghcr.io/onik/api:v1",
  status: "healthy",
  networks: [{ name: "api_default", address: "172.18.0.4" }],
  ports: ["8080:8080"],
};

const renderPage = (path = "/acme/stacks/stack-1") =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/acme/stacks/:stackId/:section?" element={<StackPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("StackPage", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    manage.stack = false;
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      if (url === "/api/stacks/stack-1/deploys") return Promise.resolve({ data: [] });
      if (url === "/api/connectors") return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
  });

  it("offers Permissions… under ⋯ for whoever may set the stack's rules, and nothing to anyone else", async () => {
    const user = userEvent.setup();
    const { unmount } = renderPage();
    expect(await screen.findByRole("heading", { name: "api" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "More actions for api" })).not.toBeInTheDocument();
    unmount();

    manage.stack = true;
    renderPage();
    await user.click(await screen.findByRole("button", { name: "More actions for api" }));
    expect(screen.getAllByRole("menuitem").map((item) => item.textContent)).toEqual(["Permissions…"]);
    await user.click(screen.getByRole("menuitem", { name: "Permissions…" }));
    const dialog = await screen.findByRole("dialog", { name: "Permissions" });
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await vi.waitFor(() => expect(screen.queryByRole("dialog", { name: "Permissions" })).not.toBeInTheDocument());
  });

  it("shows a service that is down on its row, with when, how it exited and why, and no line on a service that is up", async () => {
    const down = { id: "out-1", service: "worker", container: "api-worker-1", started_at: "2026-10-01T10:42:00Z", exit_code: 137, oom: true, restart_count: 0 };
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: { ...stack, outages: [down] } });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container, { ...container, id: "svc-2", name: "worker", container_name: "api-worker-1" }] });
      return Promise.resolve({ data: [] });
    });
    renderPage();
    const line = await screen.findByText(/^Down since \d\d:\d\d · exit 137 · out of memory$/);
    expect(line.closest("li")).toHaveTextContent("worker");
    expect(screen.getAllByText(/^Down since/)).toHaveLength(1);
  });

  it("reads the last deploy in deploy words, so a service that is down never sits beside a green Healthy", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [{ ...container, status: "exited" }] });
      if (url === "/api/stacks/stack-1/deploys") {
        return Promise.resolve({ data: [{ id: "d-1", stack_id: "stack-1", service: "api", target: "instance", image: "ghcr.io/onik/api:v1", status: "healthy", strategy: "compose", created_at: "2026-10-01T09:00:00Z", updated_at: "2026-10-01T09:01:00Z" }] });
      }
      return Promise.resolve({ data: [] });
    });
    renderPage();
    const cell = async (label: string) => within((await screen.findByText(label)).parentElement!);
    expect(await (await cell("Health")).findByText("api down")).toBeInTheDocument();
    expect(await (await cell("Last deploy")).findByText("Succeeded")).toBeInTheDocument();
    expect((await cell("Last deploy")).queryByText(/healthy/i)).not.toBeInTheDocument();
  });

  it("reads the error rate in Health: the open episode in the warning hue, a recovery in success, and nothing once the rule is off", async () => {
    const rule = { enabled: true, percent: 5, minutes: 5, min_requests: 20 };
    const episode = { id: "ep-1", stack_id: "stack-1", started_at: "2026-10-16T09:42:00Z", peak_percent: 7.24, requests: 400, errors: 29, limit_percent: 5, window_minutes: 5, min_requests: 20 };
    const serve = (extra: object) =>
      mocks.get.mockImplementation((url: string) => {
        if (url === "/api/stacks/stack-1") return Promise.resolve({ data: { ...stack, error_rate: rule, ...extra } });
        if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
        return Promise.resolve({ data: [] });
      });
    const health = async () => within((await screen.findByText("Health")).parentElement!);

    serve({ error_rate_episode: episode });
    const open = renderPage();
    expect((await health()).getByText("5xx 7.2%")).toBeInTheDocument();
    expect((await health()).getByText(/^since \d\d:\d\d$/)).toBeInTheDocument();
    expect((await health()).queryByText("No issues")).not.toBeInTheDocument();
    expect(open.container.querySelector("section[aria-label='Live status'] .bg-warning")).not.toBeNull();
    open.unmount();

    serve({ error_rate_recovered_at: "2026-10-16T09:52:00Z" });
    const recovered = renderPage();
    expect((await health()).getByText("5xx recovered")).toBeInTheDocument();
    expect(recovered.container.querySelector("section[aria-label='Live status'] .bg-success")).not.toBeNull();
    recovered.unmount();

    serve({ error_rate: { ...rule, enabled: false }, error_rate_episode: episode });
    renderPage();
    expect(await (await health()).findByText("No issues")).toBeInTheDocument();
    expect((await health()).queryByText(/^5xx/)).not.toBeInTheDocument();
  });

  it("puts Requests between Services and Deploy for a stack with a hostname, and leaves it out otherwise", async () => {
    const counted = { counted: true, from: "2026-10-16T09:43:00.000Z", requests: Array.from({ length: 60 }, () => 5), errors_5xx: Array.from({ length: 60 }, () => 0) };
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      if (url === "/api/stacks/stack-1/requests") return Promise.resolve({ data: counted });
      return Promise.resolve({ data: [] });
    });
    const { unmount } = renderPage();
    const titles = async () => (await screen.findAllByRole("heading", { level: 2 })).map((h) => h.textContent);
    await screen.findByText("requests in the last hour");
    expect((await titles()).slice(0, 3)).toEqual(["Services", "Requests", "Deploy"]);
    unmount();

    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      if (url === "/api/stacks/stack-1/requests") return Promise.resolve({ data: { counted: false, from: "", requests: [], errors_5xx: [] } });
      return Promise.resolve({ data: [] });
    });
    renderPage();
    await screen.findByRole("heading", { name: "Services" });
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/stacks/stack-1/requests"));
    expect(screen.queryByRole("heading", { name: "Requests" })).not.toBeInTheDocument();
  });

  it("an instance stack with no project has Topology as its parent crumb and offers no repository to attach", async () => {
    const gateway = { ...stack, project_id: "", name: "cloudflared-instance", slug: "cloudflared-instance", managed: false };
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: gateway });
      return Promise.resolve({ data: [] });
    });
    renderPage();

    expect(await screen.findByRole("heading", { name: "cloudflared-instance" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Topology" })).toHaveAttribute("href", "/acme/topology");
    expect(screen.queryByRole("link", { name: /attach repository/i })).not.toBeInTheDocument();
  });

  it("shows the header with the strategy and runner facts, lists the sections and lands on the overview", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "api" });
    expect(screen.getByText("compose")).toBeInTheDocument();
    expect(screen.getByText("instance")).toBeInTheDocument();
    expect(screen.getByText("No deploys yet")).toBeInTheDocument();
    expect(screen.getByText("api", { selector: "header span.font-mono" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /attach repository/i })).not.toBeInTheDocument();
    const nav = screen.getByRole("navigation", { name: "Stack sections" });
    expect(nav).toHaveTextContent("Overview");
    expect(nav).toHaveTextContent("Branch deploys");
    expect(screen.getByRole("link", { name: "Overview" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("heading", { name: "Deploy" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Services" })).toBeInTheDocument();
  });

  it("links to the logs from the nav and from each Services row for a viewer who can read them", async () => {
    access.areas = ["dns", "stackLogs"];
    renderPage();
    await screen.findByRole("heading", { name: "Services" });

    const logsLinks = await screen.findAllByRole("link", { name: "Logs" });
    expect(logsLinks.map((link) => link.getAttribute("href")).sort()).toEqual([
      "/acme/stacks/stack-1/logs",
      "/acme/stacks/stack-1/logs/api",
    ]);
  });

  it("offers no Logs entry or link without stacks:logs", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "Services" });
    await screen.findByText(/api-api-1/);

    expect(screen.queryByRole("link", { name: "Logs" })).not.toBeInTheDocument();
  });

  it("offers no image field on a compose stack without a repository, only the note that says why", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "Deploy" });
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.getByText(/has no repository to build from and nothing running to redeploy/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /redeploy/i })).not.toBeInTheDocument();
  });

  it("redeploys the running image of a run stack without a repository", async () => {
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({ data: { id: "d-9" } });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: { ...stack, strategy: "run" } });
      if (url === "/api/stacks/stack-1/services") {
        return Promise.resolve({ data: [{ ...container, image: "cloudflare/cloudflared:latest" }] });
      }
      return Promise.resolve({ data: [] });
    });
    renderPage();
    await user.click(await screen.findByRole("button", { name: "Redeploy" }));
    await vi.waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/deploys", {
        stack_id: "stack-1",
        image: "cloudflare/cloudflared:latest",
        ref: undefined,
      }),
    );
  });

  it("builds from a ref when a repository is attached", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") {
        return Promise.resolve({
          data: { ...stack, build_source: { repo_owner: "onik97", repo_name: "api", branch: "main" } },
        });
      }
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      return Promise.resolve({ data: [] });
    });
    renderPage();
    expect(await screen.findByRole("button", { name: /build & deploy/i })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /build & deploy ref/i })).toHaveValue("main");
    expect(screen.queryByRole("button", { name: "Redeploy" })).not.toBeInTheDocument();
  });

  it("hides the branch deploys section for a branch deployment", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") {
        return Promise.resolve({ data: { ...stack, derived_from: "stack-0", branch: "feature/x" } });
      }
      return Promise.resolve({ data: [] });
    });
    renderPage("/acme/stacks/stack-1/branches");
    await screen.findByRole("heading", { name: "api" });
    expect(screen.queryByRole("link", { name: "Branch deploys" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deploy" })).toBeInTheDocument();
  });

  it("hides the exposures section, and skips reading exposures, for a viewer who cannot read DNS", async () => {
    access.areas = ["projects", "topology"];
    renderPage("/acme/stacks/stack-1/exposures");
    await screen.findByRole("heading", { name: "api" });
    expect(screen.queryByRole("link", { name: "Exposures" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deploy" })).toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/dns/exposures");
  });

  it("lists each service with its image, container, hostname and addresses", async () => {
    const db = { ...container, id: "svc-2", name: "db", container_name: "api-db-1", image: "postgres:16", ports: ["5432/tcp"] };
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container, db] });
      if (url === "/api/dns/exposures") {
        return Promise.resolve({ data: [{ id: "exp-1", hostname: "app.example.com", service_id: container.id, port: 8080 }] });
      }
      return Promise.resolve({ data: [] });
    });
    renderPage();

    const api = await screen.findByRole("list", { name: "Addresses of api" });
    expect(within(api).getByRole("link", { name: "app.example.com" })).toHaveAttribute("href", "https://app.example.com");
    expect(within(api).getByText("api:8080")).toBeInTheDocument();
    expect(within(api).getByText("host :8080")).toBeInTheDocument();
    const dbAddresses = screen.getByRole("list", { name: "Addresses of db" });
    expect(within(dbAddresses).getByText("db:5432")).toBeInTheDocument();
    expect(within(dbAddresses).queryByText(/host/)).not.toBeInTheDocument();
    expect(screen.getByText("container api-db-1")).toBeInTheDocument();
    expect(screen.getByText("ghcr.io/onik/api:v1")).toBeInTheDocument();
    expect(screen.getByText("2 services on api_default", { exact: false })).toBeInTheDocument();
  });

  it("counts a compose stack's services in the header instead of naming one image", async () => {
    const stopped = { ...container, id: "svc-2", name: "worker", image: "worker:1", status: "exited" };
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container, stopped] });
      return Promise.resolve({ data: [] });
    });
    renderPage();

    const strip = await screen.findByRole("region", { name: "Live status" });
    expect(await within(strip).findByText("worker down")).toBeInTheDocument();
    expect(within(strip).getByText("1 up")).toBeInTheDocument();
    expect(screen.queryByText("Image")).not.toBeInTheDocument();
  });

  it("reads a service with an open outage as down even while the runner still sees it running", async () => {
    const outage = { id: "o-1", service: container.name, container: "c", started_at: "2026-08-12T09:00:00Z", exit_code: 137, oom: true, restart_count: 3 };
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: { ...stack, outages: [outage] } });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      return Promise.resolve({ data: [] });
    });
    renderPage();

    const strip = await screen.findByRole("region", { name: "Live status" });
    expect(await within(strip).findByText(`${container.name} down`)).toBeInTheDocument();
    expect(within(strip).getByText("0 up")).toBeInTheDocument();
    expect(within(strip).queryByText("No issues")).not.toBeInTheDocument();
  });

  describe("rollback", () => {
    const healthy = {
      id: "d-1",
      stack_id: "stack-1",
      service: "api",
      target: "instance",
      image: "",
      status: "healthy",
      strategy: "compose",
      created_at: "2026-08-12T09:00:00Z",
      updated_at: "2026-08-12T09:00:00Z",
    };
    const serve = (deploys: unknown[]) =>
      mocks.get.mockImplementation((url: string) => {
        if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
        if (url === "/api/stacks/stack-1/deploys") return Promise.resolve({ data: deploys });
        return Promise.resolve({ data: [] });
      });

    it("rolls back to the image of the last healthy deploy", async () => {
      serve([{ ...healthy, image: "ghcr.io/onik/api:v1" }]);
      renderPage();

      expect(await screen.findByRole("button", { name: "Rollback" })).toBeEnabled();
      expect(screen.getByText("Rollback re-deploys ghcr.io/onik/api:v1, the last healthy image.")).toBeInTheDocument();
    });

    it("offers no Rollback when the last healthy deploy left no image, which the server refuses", async () => {
      serve([healthy]);
      renderPage();

      await screen.findByRole("heading", { name: "api" });
      expect(screen.queryByRole("button", { name: "Rollback" })).not.toBeInTheDocument();
    });
  });

  it("shows an error display when the stack fails to load", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.reject(new Error("boom"));
      return Promise.resolve({ data: [] });
    });
    renderPage();
    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });

  it("offers to attach a repository for an unmanaged stack, linking with the stack id", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: { ...stack, managed: false } });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      return Promise.resolve({ data: [] });
    });
    renderPage();
    const link = await screen.findByRole("link", { name: /attach repository/i });
    expect(link).toHaveAttribute("href", "/acme/wizard/project/repository?stack=stack-1");
  });

  it("deletes the stack from the danger zone after confirming", async () => {
    const user = userEvent.setup();
    mocks._delete.mockResolvedValue({ data: {} });
    renderPage("/acme/stacks/stack-1/danger");

    await user.click(await screen.findByRole("button", { name: "Delete stack" }));
    const dialog = await screen.findByRole("dialog", { name: "Delete api?" });
    await user.click(within(dialog).getByRole("button", { name: "Delete stack" }));

    await vi.waitFor(() => expect(mocks._delete).toHaveBeenCalledWith("/api/stacks/stack-1"));
  });

  it("lists the hostname that will be released in the danger zone", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      if (url === "/api/dns/exposures") {
        return Promise.resolve({
          data: [
            {
              id: "exp-1",
              gateway_id: "gw-1",
              hostname: "app.example.com",
              service_id: container.id,
              service: "api",
              port: 8080,
              zone_id: "z1",
              zone: "example.com",
              created_at: "2026-08-12T09:00:00Z",
              updated_at: "2026-08-12T09:00:00Z",
            },
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });
    renderPage("/acme/stacks/stack-1/danger");
    const dangerZone = await screen.findByRole("region", { name: "Danger zone" });
    expect(within(dangerZone).getByText("app.example.com")).toBeInTheDocument();
  });

  it("renders deploy history once deploys exist", async () => {
    vi.spyOn(Date, "now").mockReturnValue(new Date("2026-08-12T12:00:00Z").getTime());
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      if (url === "/api/stacks/stack-1/deploys") {
        return Promise.resolve({
          data: [
            {
              id: "d-1",
              stack_id: "stack-1",
              service: "api",
              target: "instance",
              image: "ghcr.io/onik/api:v1",
              status: "healthy",
              strategy: "compose",
              created_at: "2026-08-12T09:00:00Z",
              updated_at: "2026-08-12T09:00:00Z",
            },
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });
    renderPage("/acme/stacks/stack-1/history");
    expect(await screen.findAllByText("ghcr.io/onik/api:v1")).not.toHaveLength(0);
    expect(within(screen.getByRole("region", { name: "Live status" })).getByText("3h ago")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deploy history" })).toBeInTheDocument();
    expect(screen.queryByText("d-1")).not.toBeInTheDocument();
  });

  it("drops the back link for a viewer without the projects area", async () => {
    access.areas = [];
    renderPage();
    expect(await screen.findByRole("heading", { name: "api" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /back to project/i })).not.toBeInTheDocument();
  });
});
