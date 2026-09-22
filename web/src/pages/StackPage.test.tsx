import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { StackPage } from "@/pages/StackPage";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), _delete: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, patch: mocks.patch, delete: mocks._delete },
  errorMessage: () => "boom",
}));

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

const renderPage = (path = "/stacks/stack-1") =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/stacks/:stackId" element={<StackPage />} />
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
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      if (url === "/api/stacks/stack-1/deploys") return Promise.resolve({ data: [] });
      if (url === "/api/connectors") return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
  });

  it("renders the stack header with the strategy and runner facts", async () => {
    renderPage();
    expect(await screen.findByRole("heading", { name: "api" })).toBeInTheDocument();
    expect(screen.getByText("compose")).toBeInTheDocument();
    expect(screen.getByText("instance")).toBeInTheDocument();
    expect(screen.getByText("No deploys yet")).toBeInTheDocument();
    expect(screen.getByText("api", { selector: "span.font-mono" })).toBeInTheDocument();
  });

  it("lists the stack sections and lands on the overview", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "api" });
    const nav = screen.getByRole("navigation", { name: "Stack sections" });
    expect(nav).toHaveTextContent("Overview");
    expect(nav).toHaveTextContent("Branch deploys");
    expect(screen.getByRole("link", { name: "Overview" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("heading", { name: "Deploy" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Services" })).toBeInTheDocument();
  });

  it("offers no image field on a compose stack without a repository, only the attach hint", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "Deploy" });
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.getByText(/attach a repository to build and deploy this stack/i)).toBeInTheDocument();
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
    renderPage("/stacks/stack-1?section=branches");
    await screen.findByRole("heading", { name: "api" });
    expect(screen.queryByRole("link", { name: "Branch deploys" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deploy" })).toBeInTheDocument();
  });

  it("renders the services table from the stack's containers", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "api" });
    expect(screen.getByText("api-api-1")).toBeInTheDocument();
    // Once in the header's Image fact, once in the table.
    expect(screen.getAllByText("ghcr.io/onik/api:v1")).toHaveLength(2);
    expect(screen.getByText("172.18.0.4", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("8080:8080")).toBeInTheDocument();
  });

  it("shows an error display when the stack fails to load", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.reject(new Error("boom"));
      return Promise.resolve({ data: [] });
    });
    renderPage();
    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });

  it("does not offer an attach-repository action for a managed stack", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "api" });
    expect(screen.queryByRole("link", { name: /attach repository/i })).not.toBeInTheDocument();
  });

  it("offers to attach a repository for an unmanaged stack, linking with the stack id", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/stacks/stack-1") return Promise.resolve({ data: { ...stack, managed: false } });
      if (url === "/api/stacks/stack-1/services") return Promise.resolve({ data: [container] });
      return Promise.resolve({ data: [] });
    });
    renderPage();
    const link = await screen.findByRole("link", { name: /attach repository/i });
    expect(link).toHaveAttribute("href", "/wizard/project/repository?stack=stack-1");
  });

  it("deletes the stack from the danger zone after confirming", async () => {
    const user = userEvent.setup();
    mocks._delete.mockResolvedValue({ data: {} });
    renderPage("/stacks/stack-1?section=danger");

    await user.click(await screen.findByRole("button", { name: "Delete stack" }));
    expect(await screen.findByRole("heading", { name: "Delete api?" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete" }));

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
    renderPage("/stacks/stack-1?section=danger");
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
    renderPage("/stacks/stack-1?section=history");
    expect(await screen.findAllByText("ghcr.io/onik/api:v1")).not.toHaveLength(0);
    expect(screen.getByText("last deploy 3h ago")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deploy history" })).toBeInTheDocument();
    expect(screen.getByText("d-1")).toBeInTheDocument();
  });
});
