import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { HistoryFieldValue } from "@/components/history/HistoryFieldValue";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

const tickets: Record<string, unknown> = {
  "/api/tickets/t-2": { id: "t-2", project_id: "p-1", number: 2, title: "Checkout" },
  "/api/tickets/t-3": { id: "t-3", project_id: "p-1", number: 3, title: "Refunds" },
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [{ id: "p-1", prefix: "ACME" }] };
    if (url in tickets) return { data: tickets[url] };
    throw Object.assign(new Error("not found"), { response: { status: 404 } });
  });
});

const renderValues = (ids: string[]) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      {ids.map((id) => (
        <HistoryFieldValue key={id} field="blocked_by" value={id} />
      ))}
    </QueryClientProvider>,
  );

describe("HistoryFieldValue", () => {
  it("reads only the tickets a change names, never the workspace's whole list", async () => {
    renderValues(["t-2", "t-3"]);
    expect(await screen.findByText("ACME-2")).toBeInTheDocument();
    expect(await screen.findByText("ACME-3")).toBeInTheDocument();
    const urls = vi.mocked(api.get).mock.calls.map(([url]) => url);
    expect(urls.filter((url) => url.startsWith("/api/tickets")).sort()).toEqual(["/api/tickets/t-2", "/api/tickets/t-3"]);
  });

  it("reads a ticket the viewer cannot open or that was deleted as another ticket", async () => {
    renderValues(["t-gone"]);
    expect(await screen.findByText("Another ticket")).toBeInTheDocument();
  });
});
