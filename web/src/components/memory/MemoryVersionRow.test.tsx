import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MemoryVersionRow } from "@/components/memory/MemoryVersionRow";
import type { MemoryVersion } from "@/models/MemoryVersion";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asOwner, asReader, settle } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
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

const memory = { id: "mem-1", project_id: "p-1" };

const renderRow = (v: MemoryVersion, isCurrent: boolean, viewer = asOwner((url) => (url === "/api/memories/mem-1" ? memory : []))) => {
  mocks.get.mockImplementation(viewer);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ul>
        <MemoryVersionRow memoryId="mem-1" version={v} isCurrent={isCurrent} />
      </ul>
    </QueryClientProvider>,
  );
};

describe("MemoryVersionRow", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.get.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  });

  it("shows the version number and author, with the title in its tooltip", () => {
    renderRow(version(), false);
    expect(screen.getByText("v2")).toBeInTheDocument();
    expect(screen.getByRole("listitem")).toHaveAttribute("title", "Deploy quirks");
    expect(screen.getByText(/onik97/)).toBeInTheDocument();
  });

  it("attributes an mcp save to the Agent via the mentioning user", () => {
    renderRow(version({ author_via: "mcp" }), false);
    expect(screen.getByText(/Agent via onik97/)).toBeInTheDocument();
  });

  it("hides revert from a reader, and locks it with the reason where their role shows locked actions", async () => {
    const reader = (showLocked: boolean) => asReader({ showLocked, permissions: ["memories:read"] }, (url) => (url === "/api/memories/mem-1" ? memory : []));
    renderRow(version(), false, reader(false));
    await settle();
    expect(screen.queryByRole("button", { name: /Revert/ })).not.toBeInTheDocument();
    cleanup();

    renderRow(version(), false, reader(true));
    const locked = await screen.findByRole("button", { name: /Revert/ });
    expect(locked).toHaveAttribute("aria-disabled", "true");
    expect(locked).toHaveAccessibleDescription("Your role can't revert memories");
  });

  it("reverts after confirming", async () => {
    mocks.post.mockResolvedValue({ data: { id: "mem-1" } });
    const user = userEvent.setup();
    renderRow(version(), false);

    await user.click(await screen.findByRole("button", { name: "Revert to v2" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Revert" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/memories/mem-1/revert", { version: 2 });
  });

  it("marks the current version instead of offering revert", () => {
    renderRow(version(), true);
    expect(screen.getByText("current")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Revert/ })).not.toBeInTheDocument();
  });
});
