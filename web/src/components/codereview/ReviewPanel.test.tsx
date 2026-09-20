import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ReviewPanel } from "@/components/codereview/ReviewPanel";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), patch: vi.fn() },
  errorMessage: vi.fn(),
}));

const review = {
  id: "r-1",
  pr_number: 42,
  repo: "acme/app",
  ticket_id: "t-1",
  doc_id: "",
  status: "approved",
  reviewer: "alice",
  created_at: "2026-08-02T12:00:00Z",
};

const renderPanel = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ReviewPanel ticketId="t-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
});

describe("ReviewPanel", () => {
  it("renders nothing with zero reviews — the rail's Development empty line covers it", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    const { container } = renderPanel();
    await vi.waitFor(() => expect(api.get).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it("lists linked pull requests with status and reviewer", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [review] });
    renderPanel();
    expect(await screen.findByText("acme/app#42")).toBeInTheDocument();
    expect(screen.getByText("approved")).toBeInTheDocument();
    expect(screen.getByText("by alice")).toBeInTheDocument();
  });

  it("omits the reviewer when there is none", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [{ ...review, reviewer: "" }] });
    renderPanel();
    expect(await screen.findByText("acme/app#42")).toBeInTheDocument();
    expect(screen.queryByText(/by /)).not.toBeInTheDocument();
  });

  it("renders nothing while reviews load — no spinner in the rail", () => {
    vi.mocked(api.get).mockImplementation(() => new Promise(() => {}));
    const { container } = renderPanel();
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the shared error display on failure", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPanel();
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Failed to load reviews")).toBeInTheDocument();
  });
});
