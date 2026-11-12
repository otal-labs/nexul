import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RepositorySearchField } from "@/components/wizard/RepositorySearchField";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "" }));

const repos = Array.from({ length: 23 }, (_, i) => ({
  id: i + 1,
  owner: "acme",
  name: `repo-${i + 1}`,
  full_name: `acme/repo-${i + 1}`,
  default_branch: "main",
  html_url: "",
  provider: "github",
}));

beforeEach(() => {
  mocks.get.mockReset();
  mocks.get.mockImplementation((url: string) =>
    Promise.resolve({ data: url === "/api/repositories" ? { repositories: repos } : {} }),
  );
  useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
});

const rows = () => screen.getAllByText(/^acme\/repo-/).length;

describe("RepositorySearchField", () => {
  it("lists ten matches, shows more on demand, and starts a new search back at ten", async () => {
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <RepositorySearchField onSelect={vi.fn()} />
      </QueryClientProvider>,
    );
    const search = screen.getByLabelText("Search repositories");

    await user.type(search, "acme");
    await screen.findByText("acme/repo-10");
    expect(rows()).toBe(10);

    await user.click(screen.getByRole("button", { name: "Show 10 more of 13" }));
    expect(rows()).toBe(20);
    await user.click(screen.getByRole("button", { name: "Show 3 more of 3" }));
    expect(rows()).toBe(23);
    expect(screen.queryByRole("button", { name: /^Show \d+ more/ })).not.toBeInTheDocument();

    await user.type(search, "/");
    await vi.waitFor(() => expect(rows()).toBe(10));
  });
});
