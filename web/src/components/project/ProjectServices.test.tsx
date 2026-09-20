import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectServices } from "@/components/project/ProjectServices";
import { api } from "@/api/client";
import type { ServiceDef } from "@/models/Service";

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

const renderServices = (list: ServiceDef[] = services) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/services") return { data: list };
    if (url === "/api/runners") return { data: runners };
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
    expect(await screen.findByRole("link", { name: /api/ })).toHaveAttribute("href", "/services/svc-1");
  });

  it("shows an empty state when there are no services", async () => {
    renderServices([]);
    expect(
      await screen.findByText("No services in this project yet — create one to define its first deploy."),
    ).toBeInTheDocument();
  });

  it("New service opens the project wizard at the repository step, project preselected", async () => {
    renderServices([]);
    expect(await screen.findByRole("link", { name: /new service/i })).toHaveAttribute(
      "href",
      "/wizard/project/repository?project=p-1",
    );
  });
});
