import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MemoryVersionRow } from "@/components/memory/MemoryVersionRow";
import type { HistoryVersionSummary } from "@/models/History";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asOwner } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: () => "error",
}));

const version = (overrides: Partial<HistoryVersionSummary> = {}): HistoryVersionSummary => ({
  version: 2,
  title: "Deploy quirks",
  created_at: "2026-09-16T12:00:00Z",
  name: "",
  editor_ids: ["onik97"],
  via: "",
  current: false,
  ...overrides,
});

const renderRow = (v: HistoryVersionSummary, onOpen = vi.fn()) => {
  mocks.get.mockImplementation(asOwner());
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ul>
        <MemoryVersionRow version={v} onOpen={onOpen} />
      </ul>
    </QueryClientProvider>,
  );
  return onOpen;
};

describe("MemoryVersionRow", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  });

  it("shows the version number and author, with the title in its tooltip", () => {
    renderRow(version());
    expect(screen.getByText("v2")).toBeInTheDocument();
    expect(screen.getByRole("listitem")).toHaveAttribute("title", "Deploy quirks");
    expect(screen.getByText(/onik97/)).toBeInTheDocument();
  });

  it("attributes an mcp save to the Agent via the mentioning user", () => {
    renderRow(version({ via: "mcp" }));
    expect(screen.getByText(/Agent via onik97/)).toBeInTheDocument();
  });

  it("opens the history on its version, current or not", async () => {
    const user = userEvent.setup();
    const onOpen = renderRow(version({ current: true }));

    expect(screen.getByText("current")).toBeInTheDocument();
    await user.click(screen.getByRole("button"));

    expect(onOpen).toHaveBeenCalledWith(2);
  });
});
