import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { HomePage } from "@/pages/HomePage";

describe("HomePage", () => {
  it("renders the product pitch and a call to action", () => {
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    );
    expect(
      screen.getByRole("heading", { name: "One button. The trail shows every step the agent took." }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign in with GitHub" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Self-host your own" })).toBeInTheDocument();
  });
});
