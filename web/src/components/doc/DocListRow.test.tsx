import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocListRow } from "@/components/doc/DocListRow";
import type { DocListItem } from "@/models/Doc";
import { usePlayRunStore } from "@/stores/playRunStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));
vi.mock("@/hooks/useDocRowActions", () => ({ useDocRowActions: () => ({}) }));
vi.mock("@/hooks/useWorkspacePath", () => ({ useWorkspacePath: () => (path: string) => path }));

const doc = (id: string): DocListItem => ({
  id,
  project_id: "p-1",
  folder_id: "f-1",
  title: `Doc ${id}`,
  version: 1,
  archived: false,
  locked: false,
  can_open: true,
  created_at: "",
  updated_at: "",
});

const running = doc("d-1");
const idle = doc("d-2");
const docs = [running, idle];

const mockApi = (active: Record<string, string>) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/docs") return { data: docs };
    if (url === "/api/plays/runs/active") return { data: { active } };
    return { data: [] };
  });

const renderRow = (d: DocListItem) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ul>
          <DocListRow doc={d} projectToken="p" selected={false} />
        </ul>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("DocListRow run indicator", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
  });

  it("shows the running indicator for a doc with an active play and asks the server about docs", async () => {
    mockApi({ "d-1": "tr-1" });
    renderRow(running);
    expect(await screen.findByRole("img", { name: "Play running" })).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith("/api/plays/runs/active", { params: { target_type: "doc", project_id: "p-1" } });
  });

  it("leaves a doc without an active play on its file icon", async () => {
    mockApi({ "d-1": "tr-1" });
    renderRow(idle);
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/plays/runs/active", expect.anything()));
    expect(screen.queryByRole("img", { name: "Play running" })).not.toBeInTheDocument();
  });

  it("drops the indicator when a live frame reports the run finished", async () => {
    mockApi({ "d-1": "tr-1" });
    renderRow(running);
    await screen.findByRole("img", { name: "Play running" });
    usePlayRunStore.getState().applyFrame({
      trail_id: "tr-1", play_id: "play-1", target_type: "doc", target_id: "d-1", state: "done", activity: null, ended_at: null, last_error: "",
    });
    await waitFor(() => expect(screen.queryByRole("img", { name: "Play running" })).not.toBeInTheDocument());
  });
});
