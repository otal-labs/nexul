import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { SidebarFooter } from "@/components/sidebar/SidebarFooter";

vi.mock("@/components/AccountMenu", () => ({ AccountMenu: () => <div data-testid="account-menu" /> }));

describe("SidebarFooter", () => {
  it("shows the account menu beside a gear that opens Your settings", () => {
    render(
      <MemoryRouter>
        <SidebarFooter collapsed={false} />
      </MemoryRouter>,
    );
    expect(screen.getByTestId("account-menu")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Your settings" })).toHaveAttribute("href", "/settings");
  });
});
