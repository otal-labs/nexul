import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { DeployRow } from "@/components/service/DeployRow";
import { DeployStatus, DeployStrategy, type Deploy } from "@/models/Stack";

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
    expect(link).toHaveAttribute("href", "/stacks/stack-1/deploys/d-1");
    expect(link).toHaveTextContent("repo build");
    expect(link).toHaveTextContent("d-1");
  });
});
