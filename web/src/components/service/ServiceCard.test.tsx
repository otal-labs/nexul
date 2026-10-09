import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ServiceCard } from "@/components/service/ServiceCard";
import { DeployStatus, DeployStrategy, type ServiceDef } from "@/models/Service";

const access = vi.hoisted(() => ({ areas: ["stacks"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

beforeEach(() => {
  access.areas = ["stacks"];
});

const baseService = (overrides: Partial<ServiceDef> = {}): ServiceDef => ({
  id: "svc-1",
  project_id: "proj-1",
  name: "api",
  target: "10.0.0.1:22",
  strategy: DeployStrategy.Compose,
  health_check: { url: "http://10.0.0.1:8080/health" },
  created_at: "",
  updated_at: "",
  ...overrides,
});

const renderCard = (service: ServiceDef, status?: DeployStatus) =>
  render(
    <MemoryRouter>
      <ServiceCard service={service} {...(status !== undefined ? { status } : {})} />
    </MemoryRouter>,
  );

describe("ServiceCard", () => {
  it("shows the build repo as the mono image line for repo-driven services", () => {
    renderCard(
      baseService({
        build_source: { repo_owner: "onik97", repo_name: "api", branch: "main" },
      }),
    );
    expect(screen.getByText("onik97/api")).toBeInTheDocument();
    expect(screen.queryByText("10.0.0.1:22")).not.toBeInTheDocument();
  });

  it("falls back to the machine, then the older target, for image-only services", () => {
    const { unmount } = renderCard(baseService({ target: "", machine: "prod-eu-west-01" }));
    expect(screen.getByText("prod-eu-west-01")).toBeInTheDocument();
    unmount();
    renderCard(baseService());
    expect(screen.getByText("10.0.0.1:22")).toBeInTheDocument();
  });

  it("renders the status badge and health dot when a status is provided", () => {
    renderCard(baseService(), DeployStatus.Healthy);
    expect(screen.getByText("healthy")).toBeInTheDocument();
  });

  it("omits the status badge and health dot without a status", () => {
    renderCard(baseService());
    expect(screen.queryByText("healthy")).not.toBeInTheDocument();
    expect(screen.queryByText("failed")).not.toBeInTheDocument();
  });

  it("links to the service detail page", () => {
    renderCard(baseService());
    expect(screen.getByRole("link", { name: /api/ })).toHaveAttribute(
      "href",
      "/acme/stacks/svc-1",
    );
  });

  it("renders a plain row, not a link, when the viewer can't read stacks", () => {
    access.areas = [];
    renderCard(baseService());
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("api")).toBeInTheDocument();
  });
});
