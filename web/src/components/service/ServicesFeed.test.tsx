import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { ServicesFeed } from "@/components/service/ServicesFeed";
import { DeployStrategy, type ServiceDef } from "@/models/Service";

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
    expect(screen.getByText("compose")).toBeInTheDocument();
    expect(screen.getByText("db")).toBeInTheDocument();
    expect(screen.getByText("10.0.0.2:22")).toBeInTheDocument();
    expect(screen.getByText("run")).toBeInTheDocument();
  });

  it("links each card to its detail page", () => {
    render(
      <MemoryRouter>
        <ServicesFeed services={services} />
      </MemoryRouter>,
    );
    expect(screen.getByRole("link", { name: /api/ })).toHaveAttribute("href", "/services/svc-1");
    expect(screen.getByRole("link", { name: /db/ })).toHaveAttribute("href", "/services/svc-2");
  });
});
