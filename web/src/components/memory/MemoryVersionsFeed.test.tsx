import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { MemoryVersionsFeed } from "@/components/memory/MemoryVersionsFeed";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asOwner } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: () => "error",
}));

const summary = (version: number) => ({
  version,
  title: "Deploy quirks",
  created_at: `2026-09-16T${String(10 + version).padStart(2, "0")}:00:00Z`,
  name: "",
  editor_ids: ["onik97"],
  via: "",
  current: false,
});

const body = (text: string) => JSON.stringify({ type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text }] }] });

const serve = (count: number) => {
  const versions = Array.from({ length: count }, (_, i) => summary(count - i));
  const first = versions[0];
  if (first) first.current = true;
  mocks.get.mockImplementation(
    asOwner((url) => {
      if (url === "/api/memories/mem-1") return { id: "mem-1", project_id: "p-1", title: "Deploy quirks" };
      if (url === "/api/memories/mem-1/history") return { versions, changes: [] };
      const picked = /\/versions\/(\d+)$/.exec(url)?.[1];
      return { version: Number(picked), title: "Deploy quirks", body: body(`text of v${picked}`) };
    }),
  );
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
};

const renderFeed = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryVersionsFeed memoryId="mem-1" />
    </QueryClientProvider>,
  );
};

describe("MemoryVersionsFeed", () => {
  it("lists the kept versions newest first from the history route and marks the current one", async () => {
    serve(2);
    renderFeed();

    await waitFor(() => expect(screen.getAllByText(/^v\d$/).map((v) => v.textContent)).toEqual(["v2", "v1"]));
    expect(screen.getByText("current")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith("/api/memories/mem-1/history");
  });

  it("shows the newest five, then opens the rest in place", async () => {
    serve(7);
    const user = userEvent.setup();
    renderFeed();

    await waitFor(() => expect(screen.getAllByText(/^v\d$/)).toHaveLength(5));
    await user.click(screen.getByRole("button", { name: "All 7 versions" }));
    expect(screen.getAllByText(/^v\d$/)).toHaveLength(7);
    expect(screen.queryByRole("button", { name: /All \d+ versions/ })).not.toBeInTheDocument();
  });

  it("opens the history dialog on the version that was clicked, where it can be restored", async () => {
    serve(3);
    const user = userEvent.setup();
    renderFeed();

    await user.click(await screen.findByRole("button", { name: /v1/ }));

    expect(await screen.findByText("Restoring adds a new version; nothing is lost")).toBeInTheDocument();
    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/memories/mem-1/versions/1"));
  });
});
