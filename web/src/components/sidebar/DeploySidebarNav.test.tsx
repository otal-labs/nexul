import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";

import { DeploySidebarNav } from "@/components/sidebar/DeploySidebarNav";
import { useSidebarStore } from "@/stores/sidebarStore";

const renderNav = (collapsed = false) =>
  render(
    <MemoryRouter>
      <DeploySidebarNav collapsed={collapsed} />
    </MemoryRouter>,
  );

beforeEach(() => {
  useSidebarStore.setState({ workspaceNavOpen: true });
});

describe("DeploySidebarNav", () => {
  it("folds down to its header and opens again", async () => {
    const user = userEvent.setup();
    renderNav();

    const header = screen.getByRole("button", { name: "Workspace" });
    expect(screen.getByRole("link", { name: /Runners/ })).toHaveAttribute("href", "/runners");

    await user.click(header);
    expect(header).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("link", { name: /Runners/ })).not.toBeInTheDocument();
    expect(useSidebarStore.getState().workspaceNavOpen).toBe(false);

    await user.click(header);
    expect(screen.getByRole("link", { name: /Runners/ })).toBeInTheDocument();
  });

  it("lists Configuration after Automations", () => {
    renderNav();
    const links = screen.getAllByRole("link").map((link) => link.getAttribute("href"));
    expect(links).toEqual(["/runners", "/topology", "/automations", "/configuration"]);
  });

  it("tags only Automations as work in progress", () => {
    renderNav();
    expect(screen.getByRole("link", { name: /Automations/ })).toHaveTextContent("WIP");
    expect(screen.getByRole("link", { name: /Topology/ })).not.toHaveTextContent("WIP");
  });

  it("collapsed rail always shows the icons, with no header to fold", () => {
    useSidebarStore.setState({ workspaceNavOpen: false });
    renderNav(true);

    expect(screen.queryByRole("button", { name: "Workspace" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Runners" })).toHaveAttribute("title", "Runners");
  });
});
