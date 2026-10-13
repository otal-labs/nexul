import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { NoComputersDisplay } from "@/components/play/NoComputersDisplay";

describe("NoComputersDisplay", () => {
  it("offers anyone adding a computer as a link to the Computers settings", () => {
    render(
      <MemoryRouter>
        <NoComputersDisplay />
      </MemoryRouter>,
    );
    expect(screen.getByRole("link", { name: "Add a computer" })).toHaveAttribute("href", "/settings/pairing");
  });
});
