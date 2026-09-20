import { render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { TopologyPage } from "@/pages/TopologyPage";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

describe("TopologyPage", () => {
  it("renders the React Flow canvas", () => {
    mocks.get.mockResolvedValue({ data: { schema_version: 2, nodes: [], edges: [] } });
    const { container } = render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <TopologyPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(container.querySelector(".react-flow")).toBeInTheDocument();
  });
});
