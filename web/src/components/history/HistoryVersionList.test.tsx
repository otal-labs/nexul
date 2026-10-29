import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { HistoryVersionList } from "@/components/history/HistoryVersionList";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { HistoryFieldChange, HistoryVersionSummary } from "@/models/History";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() } }));

const version = (n: number, day: number, current = false): HistoryVersionSummary => ({
  version: n,
  created_at: `2026-10-0${day}T10:00:00Z`,
  name: "",
  editor_ids: ["u-1"],
  via: "",
  current,
});

const change: HistoryFieldChange = {
  id: "c-1",
  field: "status",
  from: "In progress",
  to: "Testing",
  created_at: "2026-10-02T12:00:00Z",
  actor: { user_id: "u-1" },
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockResolvedValue({
    data: {
      people: [
        { user_id: "u-1", login: "alice", display_name: "Alice Moreau", avatar_url: "" },
        { user_id: "u-2", login: "bob", display_name: "Bob Tanaka", avatar_url: "" },
      ],
    },
  });
});

describe("HistoryVersionList", () => {
  it("shows a field change between the versions it happened among, as a line that cannot be picked", () => {
    const client = new QueryClient();
    render(
      <QueryClientProvider client={client}>
        <HistoryVersionList versions={[version(2, 3, true), version(1, 1)]} changes={[change]} selected={2} onSelect={vi.fn()} />
      </QueryClientProvider>,
    );

    const items = screen.getAllByRole("listitem");
    expect(items[1]).toHaveTextContent("Status");
    expect(items[1]).toHaveTextContent("Testing");
    expect(screen.getAllByRole("button")).toHaveLength(2);
    expect(screen.getByText("Testing").closest("[aria-disabled]")).toHaveAttribute("aria-disabled", "true");
  });

  it("shows extra editors as small avatars, never as a +N that reads like a size, and names them all in the row's name", async () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <HistoryVersionList versions={[{ ...version(2, 3, true), editor_ids: ["u-1", "u-2"] }, version(1, 1)]} selected={2} onSelect={vi.fn()} />
      </QueryClientProvider>,
    );

    expect(await screen.findByRole("button", { name: /Alice Moreau, Bob Tanaka/ })).toBeInTheDocument();
    expect(screen.queryByText(/^\+\d/)).not.toBeInTheDocument();
  });

  it("says 'plus named ones' only where a version can be named: docs always, others only once one is kept", () => {
    const render_ = (props: { namesVersions?: boolean; named?: boolean }) =>
      render(
        <QueryClientProvider client={new QueryClient()}>
          <HistoryVersionList
            versions={[version(2, 3, true), { ...version(1, 1), name: props.named ? "Q3 freeze" : "" }]}
            selected={2}
            {...(props.namesVersions !== undefined && { namesVersions: props.namesVersions })}
            onSelect={vi.fn()}
          />
        </QueryClientProvider>,
      );

    const first = render_({});
    expect(screen.getByText("Only the last 10 versions are kept, plus named ones.")).toBeInTheDocument();
    first.unmount();
    const second = render_({ namesVersions: false });
    expect(screen.getByText("Only the last 10 versions are kept.")).toBeInTheDocument();
    second.unmount();
    render_({ namesVersions: false, named: true });
    expect(screen.getByText("Only the last 10 versions are kept, plus named ones.")).toBeInTheDocument();
  });

  describe("below 1024px", () => {
    afterEach(() => {
      // @ts-expect-error jsdom has no matchMedia; the hook reads absence as a wide window.
      delete window.matchMedia;
    });

    it("swaps the 'via an agent' words for a glyph so the author's name keeps its width, the phrase staying in the row's name", () => {
      window.matchMedia = vi.fn().mockReturnValue({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() });
      render(
        <QueryClientProvider client={new QueryClient()}>
          <HistoryVersionList versions={[{ ...version(2, 3, true), via: "mcp" }, version(1, 1)]} selected={2} onSelect={vi.fn()} />
        </QueryClientProvider>,
      );

      expect(screen.queryByText("via an agent")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: /via an agent/ })).toBeInTheDocument();
    });
  });
});
