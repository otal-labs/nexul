import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocTicketsSection } from "@/components/doc/DocTicketsSection";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

const renderSection = (docId = "doc-1") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DocTicketsSection docId={docId} />
    </QueryClientProvider>,
  );
};

const ticket = (id: string, title: string) => ({
  id,
  project_id: "p-1",
  title,
  body: "",
  status: "open",
  doc_id: "doc-1",
  assignee: "",
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
});

beforeEach(() => {
  vi.mocked(api.get).mockReset();
});

describe("DocTicketsSection", () => {
  it("shows the shared loading display while fetching", () => {
    vi.mocked(api.get).mockReturnValue(new Promise(() => {}));
    renderSection();
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("shows the shared error display when the fetch fails", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderSection();
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("boom")).toBeInTheDocument();
  });

  it("renders nothing when the doc has no tickets", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    const { container } = renderSection();
    await vi.waitFor(() => {
      expect(screen.queryByRole("status")).not.toBeInTheDocument();
    });
    expect(screen.queryByText("Tickets from this doc")).not.toBeInTheDocument();
    expect(container).toBeEmptyDOMElement();
  });

  it("lists the tickets derived from the doc", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [ticket("t-1", "Write migrations"), ticket("t-2", "Wire FTS")] });
    renderSection();
    expect(await screen.findByText("Tickets from this doc")).toBeInTheDocument();
    expect(screen.getByText("Write migrations")).toBeInTheDocument();
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
  });
});
