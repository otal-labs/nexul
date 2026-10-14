import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DevelopmentSection } from "@/components/ticket/DevelopmentSection";
import type { TicketLinks } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asViewer, memberWho } from "@/test/viewer";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const perms = vi.hoisted(() => ({ canWrite: true }));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const links = {
  prs: [
    { owner: "acme", repo: "app", number: 42, title: "Fix login", sha: "abc", state: "open" },
    { owner: "acme", repo: "app", number: 43, title: "Fix auth", sha: "def", state: "merged" },
  ],
  branches: [{ owner: "acme", repo: "app", branch: "ticket/1" }],
};

// The viewer's permission reads answer as a member who may or may not write tickets; every other read gets `data`.
const serve = (data: unknown) =>
  vi.mocked(api.get).mockImplementation(asViewer(memberWho(perms.canWrite ? ["tickets:write"] : [], false), () => data));

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
  vi.mocked(api.delete).mockReset();
  perms.canWrite = true;
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("DevelopmentSection", () => {
  it("shows an empty state with no links", async () => {
    serve({ prs: [], branches: [] });
    renderSection();
    expect(await screen.findByText("No branches or PRs linked yet.")).toBeInTheDocument();
  });

  it("tolerates null link arrays from a stale backend shape", async () => {
    serve({ prs: null, branches: null } as unknown as TicketLinks);
    renderSection();
    expect(await screen.findByText("No branches or PRs linked yet.")).toBeInTheDocument();
  });

  it("lists linked branches and prs under their repo", async () => {
    serve(links);
    renderSection();
    expect(await screen.findByRole("link", { name: "acme/app" })).toHaveAttribute("href", "https://github.com/acme/app");
    expect(screen.getByRole("link", { name: "ticket/1" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "#42 Fix login, open" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "#43 Fix auth, merged" })).toBeInTheDocument();
  });

  it("links an existing branch", async () => {
    const user = userEvent.setup();
    serve({ prs: [], branches: [] });
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
    serve({ prs: [], branches: [] });
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

  it("unlinks a branch and shows the links the server answers with", async () => {
    const user = userEvent.setup();
    serve(links);
    vi.mocked(api.delete).mockResolvedValue({ data: { prs: links.prs, branches: [] } });
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Unlink branch ticket/1" }));

    expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1/branches", { params: { owner: "acme", repo: "app", branch: "ticket/1" } });
    await waitFor(() => expect(screen.queryByRole("link", { name: "ticket/1" })).not.toBeInTheDocument());
    expect(screen.getByRole("link", { name: "#42 Fix login, open" })).toBeInTheDocument();
  });

  it("unlinks a pull request and keeps the others", async () => {
    const user = userEvent.setup();
    serve(links);
    vi.mocked(api.delete).mockResolvedValue({ data: { prs: [links.prs[1]], branches: links.branches } });
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Unlink pull request #42" }));

    expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1/prs", { params: { owner: "acme", repo: "app", number: 42 } });
    await waitFor(() => expect(screen.queryByRole("link", { name: "#42 Fix login, open" })).not.toBeInTheDocument());
    expect(screen.getByRole("link", { name: "#43 Fix auth, merged" })).toBeInTheDocument();
  });

  it("offers no unlink to someone who cannot write tickets", async () => {
    perms.canWrite = false;
    serve(links);
    renderSection();

    expect(await screen.findByRole("link", { name: "ticket/1" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Unlink/ })).not.toBeInTheDocument();
  });

  it("shows an error state", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderSection();
    expect(await screen.findByText("Couldn't load development links.")).toBeInTheDocument();
  });
});
