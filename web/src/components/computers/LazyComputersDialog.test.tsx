import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { LazyComputersDialog } from "@/components/computers/LazyComputersDialog";

vi.mock("@/components/computers/ComputersDialog", () => ({ ComputersDialog: () => <p>Computers dialog</p> }));

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <LazyComputersDialog />
    </MemoryRouter>,
  );

describe("the Computers dialog in the app shell", () => {
  it("loads and opens on any page whose address carries the computers param, even an empty one", async () => {
    renderAt("/acme/board?computers=");
    expect(await screen.findByText("Computers dialog")).toBeInTheDocument();
  });

  it("loads nothing while the param is absent", () => {
    renderAt("/acme/board");
    expect(screen.queryByText("Computers dialog")).not.toBeInTheDocument();
  });
});
