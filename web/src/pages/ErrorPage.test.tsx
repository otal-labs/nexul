import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, RouterProvider } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ErrorPage } from "@/pages/ErrorPage";

const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return { ...actual, useNavigate: () => mockNavigate };
});

describe("ErrorPage", () => {
  it("renders the not-found title and a back link", () => {
    const router = createMemoryRouter(
      [
        { path: "/", element: <div>home</div> },
        { path: "*", element: <ErrorPage /> },
      ],
      { initialEntries: ["/missing"] },
    );
    render(<RouterProvider router={router} />);
    expect(screen.getByRole("heading", { name: "Page not found" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back home" })).toBeInTheDocument();
  });

  it("navigates back when Go back is clicked", async () => {
    const user = userEvent.setup();
    const router = createMemoryRouter(
      [
        { path: "/", element: <div>home</div> },
        { path: "*", element: <ErrorPage /> },
      ],
      { initialEntries: ["/", "/missing"], initialIndex: 1 },
    );
    render(<RouterProvider router={router} />);

    await user.click(screen.getByRole("button", { name: "Go back" }));

    expect(mockNavigate).toHaveBeenCalledWith(-1);
  });

  it("renders a reload button when reached via errorElement (thrown error, e.g. a stale chunk)", () => {
    const Thrower = () => {
      throw new Error("Failed to fetch dynamically imported module");
    };
    const router = createMemoryRouter(
      [{ path: "/", element: <Thrower />, errorElement: <ErrorPage /> }],
      { initialEntries: ["/"] },
    );
    render(<RouterProvider router={router} />);
    expect(screen.getByRole("heading", { name: "Something went wrong", level: 1 })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reload" })).toBeInTheDocument();
  });
});
