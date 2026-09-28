import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SidebarFooter } from "@/components/sidebar/SidebarFooter";

vi.mock("@/components/AccountMenu", () => ({ AccountMenu: () => <div data-testid="account-menu" /> }));
vi.mock("@/components/ThemeToggle", () => ({ ThemeToggle: () => <div data-testid="theme-toggle" /> }));

describe("SidebarFooter", () => {
  it("shows the account menu beside the theme toggle when logged in", () => {
    render(<SidebarFooter collapsed={false} isLoggedIn />);
    expect(screen.getByTestId("account-menu")).toBeInTheDocument();
    expect(screen.getByTestId("theme-toggle")).toBeInTheDocument();
  });

  it("shows only the theme toggle when logged out", () => {
    render(<SidebarFooter collapsed isLoggedIn={false} />);
    expect(screen.queryByTestId("account-menu")).not.toBeInTheDocument();
    expect(screen.getByTestId("theme-toggle")).toBeInTheDocument();
  });
});
