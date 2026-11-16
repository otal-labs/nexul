import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { SidebarHeader } from "@/components/sidebar/SidebarHeader";

vi.mock("@/hooks/VersionHooks", () => ({ useServerVersion: () => ({ data: { version: "v0.2.0-beta-004", update_available: false } }) }));
vi.mock("@/ee/hooks/BrandingHooks", () => ({ useLook: () => ({ data: undefined }) }));
vi.mock("@/components/sidebar/UpdateButton", () => ({ UpdateButton: () => null }));

describe("SidebarHeader", () => {
  it("opens the server version from the Beta pill on a tap, not only on hover", async () => {
    render(
      <MemoryRouter>
        <SidebarHeader collapsed={false} onToggleCollapse={() => {}} />
      </MemoryRouter>,
    );

    await userEvent.click(screen.getByRole("button", { name: "Beta" }));

    expect(await screen.findByText("v0.2.0-beta-004")).toBeVisible();
  });
});
