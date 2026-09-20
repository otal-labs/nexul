import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MemoryVersionRow } from "@/components/memory/MemoryVersionRow";
import type { MemoryVersion } from "@/models/MemoryVersion";

const mocks = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const version = (overrides: Partial<MemoryVersion> = {}): MemoryVersion => ({
  id: "ver-1",
  memory_id: "mem-1",
  version: 2,
  title: "Deploy quirks",
  when_to_use: "when deploying",
  body: "body",
  always_included: false,
  author_id: "onik97",
  author_via: "",
  created_at: "2026-09-16T12:00:00Z",
  ...overrides,
});

const renderRow = (v: MemoryVersion, isCurrent: boolean, canRevert: boolean) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ul>
        <MemoryVersionRow memoryId="mem-1" version={v} isCurrent={isCurrent} canRevert={canRevert} />
      </ul>
    </QueryClientProvider>,
  );
};

describe("MemoryVersionRow", () => {
  beforeEach(() => {
    mocks.post.mockReset();
  });

  it("shows the version number, title, and author", () => {
    renderRow(version(), false, true);
    expect(screen.getByText("v2")).toBeInTheDocument();
    expect(screen.getByText("Deploy quirks")).toBeInTheDocument();
    expect(screen.getByText(/onik97/)).toBeInTheDocument();
  });

  it("attributes an mcp save to the Agent via the mentioning user", () => {
    renderRow(version({ author_via: "mcp" }), false, true);
    expect(screen.getByText(/Agent via onik97/)).toBeInTheDocument();
  });

  it("hides revert for the current version", () => {
    renderRow(version(), true, true);
    expect(screen.queryByRole("button", { name: /Revert/ })).not.toBeInTheDocument();
  });

  it("hides revert when the caller lacks memories:write", () => {
    renderRow(version(), false, false);
    expect(screen.queryByRole("button", { name: /Revert/ })).not.toBeInTheDocument();
  });

  it("reverts after confirming", async () => {
    mocks.post.mockResolvedValue({ data: { id: "mem-1" } });
    const user = userEvent.setup();
    renderRow(version(), false, true);

    await user.click(screen.getByRole("button", { name: /Revert/ }));
    await user.click(await screen.findByRole("button", { name: "Confirm" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/memories/mem-1/revert", { version: 2 });
  });
});
