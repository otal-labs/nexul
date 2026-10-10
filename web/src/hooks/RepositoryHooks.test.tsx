import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { repositoryFollower, useSearchRepositories } from "@/hooks/RepositoryHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { followFrame, isStale, seeded } from "@/test/followFrame";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() } }));

describe("useSearchRepositories", () => {
  it("asks the server past its cache when a loaded search is refetched, so a newly installed account shows up", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { repositories: [] } });
    useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    renderHook(() => useSearchRepositories(" onik "), { wrapper });
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(1));
    expect(vi.mocked(api.get).mock.calls[0]![1]).toEqual(expect.objectContaining({ params: { workspace_id: "ws-1", q: "onik" } }));

    await client.refetchQueries({ queryKey: ["repositories"] });
    expect(vi.mocked(api.get).mock.calls[1]![1]).toEqual(expect.objectContaining({ params: { workspace_id: "ws-1", q: "onik", refresh: 1 } }));
  });
});

describe("repositoryFollower", () => {
  it("refreshes the installations card and leaves the repository lists, which are each person's own GitHub view", async () => {
    const client = seeded([
      [["repository-installations"], []],
      [["repositories", "ws-acme", "api"], []],
      [["repositories", "ws-globex", "api"], []],
    ]);
    await followFrame(repositoryFollower, "repository.installation.unassigned", { account_id: 11, account_login: "acme", workspace_id: "ws-acme" }, client);
    expect(isStale(client, ["repository-installations"])).toBe(true);
    expect(isStale(client, ["repositories", "ws-acme", "api"])).toBe(false);
    expect(isStale(client, ["repositories", "ws-globex", "api"])).toBe(false);
  });
});
