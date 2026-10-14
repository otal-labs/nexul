import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnersPage } from "@/pages/RunnersPage";
import { resetViewer, viewer } from "@/test/viewerActions";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);

const renderPage = () => {
  mocks.get.mockImplementation((url: string) =>
    Promise.resolve({ data: url === "/api/runners/latest-version" ? { version: "v0.1.6" } : [] }),
  );
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient()}>
        <RunnersPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
};

describe("RunnersPage", () => {
  beforeEach(resetViewer);

  it("renders the page heading and panels, with Add runner for someone who may add machines", async () => {
    viewer.instance = ["machines:create"];
    renderPage();
    expect(screen.getByRole("heading", { level: 1, name: "Runners" })).toBeInTheDocument();
    expect(await screen.findByText("Waiting for a runner")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Add runner/ })).toBeInTheDocument();
  });

  it.each([
    ["hides Add runner", false],
    ["locks Add runner", true],
  ])("%s from someone who may not add machines", async (_name, showLocked) => {
    viewer.showLocked = showLocked;
    renderPage();
    expect(await screen.findByText("Waiting for a runner")).toBeInTheDocument();

    const add = screen.queryByRole("button", { name: /Add runner/ });
    expect(!!add).toBe(showLocked);
    expect(add === null || add.getAttribute("aria-disabled") === "true").toBe(true);
  });
});
