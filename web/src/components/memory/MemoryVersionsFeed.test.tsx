import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { MemoryVersionsFeed } from "@/components/memory/MemoryVersionsFeed";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: () => "error",
}));

const versions = [
  {
    id: "ver-2",
    memory_id: "mem-1",
    version: 2,
    title: "Deploy quirks",
    when_to_use: "",
    body: "v2",
    always_included: false,
    author_id: "onik97",
    author_via: "",
    created_at: "2026-09-16T13:00:00Z",
  },
  {
    id: "ver-1",
    memory_id: "mem-1",
    version: 1,
    title: "Deploy quirks",
    when_to_use: "",
    body: "v1",
    always_included: false,
    author_id: "onik97",
    author_via: "",
    created_at: "2026-09-16T12:00:00Z",
  },
];

const renderFeed = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryVersionsFeed memoryId="mem-1" currentVersion={2} canRevert />
    </QueryClientProvider>,
  );
};

describe("MemoryVersionsFeed", () => {
  it("lists every version newest first, and only offers revert on non-current rows", async () => {
    mocks.get.mockResolvedValue({ data: versions });
    renderFeed();

    await waitFor(() => expect(screen.getAllByText(/^v\d$/)).toHaveLength(2));
    const buttons = screen.getAllByRole("button", { name: /Revert/ });
    expect(buttons).toHaveLength(1); // only v1, not the current v2
    expect(mocks.get).toHaveBeenCalledWith("/api/memories/mem-1/versions");
  });
});
