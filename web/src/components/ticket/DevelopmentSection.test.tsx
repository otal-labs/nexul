import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DevelopmentSection } from "@/components/ticket/DevelopmentSection";
import type { TicketLinks } from "@/models/Ticket";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const links = {
  prs: [
    { owner: "acme", repo: "app", number: 42, title: "Fix login", sha: "abc", state: "open" },
    { owner: "acme", repo: "app", number: 43, title: "Fix auth", sha: "def", state: "merged" },
  ],
  branches: [{ owner: "acme", repo: "app", branch: "ticket/1" }],
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DevelopmentSection ticketId="t-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("DevelopmentSection", () => {
  it("shows an empty state with no links", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { prs: [], branches: [] } });
    renderSection();
    expect(await screen.findByText("No branches or PRs linked yet.")).toBeInTheDocument();
  });

  it("tolerates null link arrays from a stale backend shape", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { prs: null, branches: null },
    } as unknown as { data: TicketLinks });
    renderSection();
    expect(await screen.findByText("No branches or PRs linked yet.")).toBeInTheDocument();
  });

  it("lists linked prs with state and branches", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: links });
    renderSection();
    expect(await screen.findByText("acme/app#42")).toBeInTheDocument();
    expect(screen.getByText("acme/app#43")).toBeInTheDocument();
    expect(screen.getByText("open")).toBeInTheDocument();
    expect(screen.getByText("merged")).toBeInTheDocument();
    expect(screen.getByText("acme/app:ticket/1")).toBeInTheDocument();
  });

  it("links an existing branch", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockResolvedValue({ data: { prs: [], branches: [] } });
    vi.mocked(api.post).mockResolvedValue({ data: { prs: [], branches: [{ owner: "acme", repo: "app", branch: "feat" }] } });
    renderSection();

    await screen.findByText("No branches or PRs linked yet.");
    await user.click(screen.getByRole("button", { name: "Link development item" }));
    await user.click(await screen.findByRole("button", { name: "Link branch" }));
    await user.type(screen.getByLabelText("Branch owner"), "acme");
    await user.type(screen.getByLabelText("Branch repo"), "app");
    await user.type(screen.getByLabelText("Branch name"), "feat");
    await user.click(screen.getByRole("button", { name: "Link" }));

    expect(api.post).toHaveBeenCalledWith("/api/tickets/t-1/branches", {
      owner: "acme",
      repo: "app",
      branch: "feat",
    });
  });

  it("links an existing pull request", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockResolvedValue({ data: { prs: [], branches: [] } });
    vi.mocked(api.post).mockResolvedValue({ data: { prs: [], branches: [] } });
    renderSection();

    await screen.findByText("No branches or PRs linked yet.");
    await user.click(screen.getByRole("button", { name: "Link development item" }));
    await user.click(await screen.findByRole("button", { name: "Link PR" }));
    await user.type(screen.getByLabelText("PR owner"), "acme");
    await user.type(screen.getByLabelText("PR repo"), "app");
    await user.type(screen.getByLabelText("PR number"), "99");
    await user.click(screen.getByRole("button", { name: "Link" }));

    expect(api.post).toHaveBeenCalledWith("/api/tickets/t-1/prs", {
      owner: "acme",
      repo: "app",
      number: 99,
    });
  });

  it("shows an error state", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderSection();
    expect(await screen.findByText("Failed to load development links.")).toBeInTheDocument();
  });
});
