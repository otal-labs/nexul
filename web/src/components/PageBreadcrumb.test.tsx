import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { PageBreadcrumb } from "@/components/PageBreadcrumb";

const long = "Globex International Logistics, Supply Chain and Warehousing Operations Division";

const renderCrumbs = (labels: string[]) =>
  render(
    <MemoryRouter>
      <PageBreadcrumb crumbs={labels.map((label) => ({ label, to: "/" }))} />
    </MemoryRouter>,
  );

describe("PageBreadcrumb", () => {
  it("cuts a long middle crumb on whole characters with no space before its ellipsis, and keeps the full name on hover", () => {
    renderCrumbs([long, "Configuration"]);

    const crumb = screen.getByRole("link", { name: "Globex International Logistics, Supply…" });
    expect(crumb.closest("li")).toHaveAttribute("title", long);
  });

  it("renders a crumb whose label is still loading instead of throwing", () => {
    render(
      <MemoryRouter>
        <PageBreadcrumb crumbs={[{ label: undefined as unknown as string, to: "/" }, { label: "Configuration" }]} />
      </MemoryRouter>,
    );

    expect(screen.getByText("Configuration")).toBeInTheDocument();
  });

  it("leaves a short crumb and the last crumb whole", () => {
    renderCrumbs(["Northwind Industries", long]);

    expect(screen.getByRole("link", { name: "Northwind Industries" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: long })).toBeInTheDocument();
  });
});
