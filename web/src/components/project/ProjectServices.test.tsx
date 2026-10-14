import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectServices } from "@/components/project/ProjectServices";
import { api } from "@/api/client";
import type { ServiceDef } from "@/models/Service";

const access = vi.hoisted(() => ({ areas: ["stacks", "editProjects", "editStacks", "editDeploys"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

beforeEach(() => {
  access.areas = ["stacks", "editProjects", "editStacks", "editDeploys"];
});

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const services: ServiceDef[] = [
  { id: "svc-1", project_id: "p-1", name: "api", target: "10.0.0.1:22", strategy: "compose" } as ServiceDef,
];

const runners = [
  { id: "r1", name: "instance", connected: true, last_seen: "2026-09-03T00:00:00Z", running_job: null, version: "0.1.0" },
];

const repo = (full: string, role = "app") => {
  const [owner, name] = full.split("/");
  return { owner, name, full_name: full, connector_id: "github", role };
};

const renderServices = (list: ServiceDef[] = services, repos: unknown[] = []) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/services") return { data: list };
    if (url === "/api/runners") return { data: runners };
    if (url === "/api/projects/p-1/repos") return { data: repos };
    return { data: [] };
  });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter>
        <ProjectServices projectId="p-1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("ProjectServices", () => {
  it("lists services scoped to the project", async () => {
    renderServices();
    expect(await screen.findByRole("link", { name: /api/ })).toHaveAttribute("href", "/acme/stacks/svc-1");
  });

  it("shows an empty state when there are no services", async () => {
    renderServices([]);
    expect(
      await screen.findByText("No services yet. Create one to set up its first deploy."),
    ).toBeInTheDocument();
  });

  it("New service opens the project wizard at the repository step, project preselected, as a service beside any setup", async () => {
    renderServices([]);
    expect(await screen.findByRole("link", { name: /new service/i })).toHaveAttribute(
      "href",
      "/acme/wizard/project/repository?project=p-1&add=1",
    );
  });

  it("offers to deploy each attached repository no service builds from, never the tests repository", async () => {
    const deployed = { ...services[0]!, build_source: { repo_owner: "acme", repo_name: "api" } };
    renderServices([deployed], [repo("acme/api"), repo("acme/portal"), repo("acme/e2e", "tests")]);

    expect(await screen.findByRole("link", { name: /deploy this repository/i })).toHaveAttribute(
      "href",
      "/acme/wizard/project/repository?project=p-1&add=1&repo=acme%2Fportal",
    );
    expect(screen.getByText("acme/portal")).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: /deploy this repository/i })).toHaveLength(1);
  });

  it("puts a waiting repository where the empty line was", async () => {
    renderServices([], [repo("acme/portal")]);
    expect(await screen.findByText("Attached, not deployed yet")).toBeInTheDocument();
    expect(screen.queryByText(/No services yet/)).not.toBeInTheDocument();
  });

  it("shows no repository as waiting to someone who can't read stacks, whose service list is always empty", async () => {
    access.areas = ["editProjects", "editStacks", "editDeploys"];
    renderServices([], [repo("acme/portal")]);
    expect(await screen.findByText(/No services yet/)).toBeInTheDocument();
    expect(screen.queryByText("Attached, not deployed yet")).not.toBeInTheDocument();
  });

  it("hides New service and unlinks the cards without projects:write and stacks:read", async () => {
    access.areas = [];
    renderServices();
    expect(await screen.findByText("api")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
