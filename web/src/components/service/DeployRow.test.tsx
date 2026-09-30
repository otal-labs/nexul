import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DeployRow } from "@/components/service/DeployRow";
import { DeployStatus, DeployStrategy, type Deploy } from "@/models/Stack";

const access = vi.hoisted(() => ({ areas: ["deploys"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

beforeEach(() => {
  access.areas = ["deploys"];
});

const deploy: Deploy = {
  id: "d-1",
  kind: "build",
  stack_id: "stack-1",
  service: "api",
  target: "instance",
  image: "",
  status: DeployStatus.Running,
  strategy: DeployStrategy.Compose,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe("DeployRow", () => {
  it("links the row to the deploy's log page", () => {
    render(
      <MemoryRouter>
        <ul>
          <DeployRow deploy={deploy} />
        </ul>
      </MemoryRouter>,
    );
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/acme/stacks/stack-1/deploys/d-1");
    expect(link).toHaveTextContent("repo build");
    expect(link).toHaveTextContent("d-1");
  });

  it("renders a plain row, not a link, when the viewer can't read deploys", () => {
    access.areas = [];
    render(
      <MemoryRouter>
        <ul>
          <DeployRow deploy={deploy} />
        </ul>
      </MemoryRouter>,
    );
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("repo build")).toBeInTheDocument();
  });
});
