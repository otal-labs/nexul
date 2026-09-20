import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { RunnersPage } from "@/pages/RunnersPage";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

describe("RunnersPage", () => {
  it("renders the page heading and panels", async () => {
    mocks.get.mockImplementation((url: string) =>
      Promise.resolve({ data: url === "/api/runners/latest-version" ? { version: "v0.1.6" } : [] }),
    );
    render(
      <QueryClientProvider client={new QueryClient()}>
        <RunnersPage />
      </QueryClientProvider>,
    );
    expect(screen.getByRole("heading", { level: 1, name: "Runners" })).toBeInTheDocument();
    expect(await screen.findByText("Waiting for a runner")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Add runner/ })).toBeInTheDocument();
  });
});
