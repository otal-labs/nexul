import { render, screen } from "@testing-library/react";
import { ShieldCheck } from "lucide-react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { SetupNavLink } from "@/components/sidebar/SetupNavLink";

const renderLink = (collapsed: boolean, path = "/") =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <SetupNavLink to="/acme/access" label="Set up access" icon={ShieldCheck} collapsed={collapsed} />
    </MemoryRouter>,
  );

describe("SetupNavLink", () => {
  it("names the pending setup on the icon rail, where only a dot says so", () => {
    renderLink(true);
    expect(screen.getByRole("link", { name: "Set up access, setup pending" })).toHaveAttribute("href", "/acme/access");
  });

  it("is the current page's link like any other nav row", () => {
    renderLink(false, "/acme/access");
    expect(screen.getByRole("link", { name: "Set up access" })).toHaveAttribute("aria-current", "page");
  });
});
