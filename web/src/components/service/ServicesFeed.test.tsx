import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ServicesFeed } from "@/components/service/ServicesFeed";
import { DeployStrategy, type ServiceDef } from "@/models/Service";

const access = vi.hoisted(() => ({ areas: ["stacks"] as string[] }));
const manage = vi.hoisted(() => ({ stack: false }));
vi.mock("@/hooks/useCanManageAccess", () => ({ useCanManageAccess: (entity?: { type: string }) => entity?.type === "stack" && manage.stack }));
const history = vi.hoisted(() => ({ deploys: undefined as unknown[] | undefined }));
vi.mock("@/hooks/StackHooks", () => ({ useFetchStackDeploys: () => ({ data: history.deploys }) }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

const services: ServiceDef[] = [
  {
    id: "svc-1",
    project_id: "proj-1",
    name: "api",
    target: "10.0.0.1:22",
    strategy: DeployStrategy.Compose,
    health_check: { url: "http://10.0.0.1:8080/health" },
    created_at: "",
    updated_at: "",
  },
  {
    id: "svc-2",
    project_id: "proj-1",
    name: "db",
    target: "10.0.0.2:22",
    strategy: DeployStrategy.Run,
    health_check: { url: "http://10.0.0.2:5432/health" },
    created_at: "",
    updated_at: "",
  },
];

describe("ServicesFeed", () => {
  it("renders one card per service with name, target, and strategy", () => {
    render(
      <MemoryRouter>
        <ServicesFeed services={services} />
      </MemoryRouter>,
    );
    expect(screen.getByText("api")).toBeInTheDocument();
    expect(screen.getByText("10.0.0.1:22")).toBeInTheDocument();
    expect(screen.getByLabelText("compose")).toBeInTheDocument();
    expect(screen.getByText("db")).toBeInTheDocument();
    expect(screen.getByText("10.0.0.2:22")).toBeInTheDocument();
    expect(screen.getByLabelText("run")).toBeInTheDocument();
  });

  it("leaves a machine every service shares to the header, and keeps the repository", () => {
    const shared = services.map((svc) => ({ ...svc, target: "", machine: "prod-1", build_source: { repo_owner: "acme", repo_name: "api" } }));
    render(
      <MemoryRouter>
        <ServicesFeed services={shared as ServiceDef[]} />
      </MemoryRouter>,
    );
    expect(screen.queryByText("prod-1")).not.toBeInTheDocument();
    expect(screen.getAllByText("acme/api")).toHaveLength(2);
  });
});
