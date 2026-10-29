import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { StackNav } from "@/components/stack/StackNav";

describe("StackNav", () => {
  it("links Alerts to the alerts section under the stack's settings", () => {
    render(
      <MemoryRouter>
        <StackNav stackId="stack-1" active="environment" showBranches showExposures />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Alerts" })).toHaveAttribute("href", "/acme/stacks/stack-1/settings/alerts");
  });
});
