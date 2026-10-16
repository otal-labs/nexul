import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { StackNav } from "@/components/stack/StackNav";

describe("StackNav", () => {
  it("puts Alerts right after Logs", () => {
    render(
      <MemoryRouter>
        <StackNav stackId="stack-1" active="overview" showBranches showExposures showLogs />
      </MemoryRouter>,
    );

    const labels = screen.getAllByRole("link").map((link) => link.textContent);
    expect(labels.slice(labels.indexOf("Logs"), labels.indexOf("Logs") + 2)).toEqual(["Logs", "Alerts"]);
    expect(screen.getByRole("link", { name: "Alerts" })).toHaveAttribute("href", "/acme/stacks/stack-1/alerts");
  });
});
