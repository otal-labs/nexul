import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MemoriesListPane } from "@/components/memory/MemoriesListPane";
import type { Memory } from "@/models/Memory";
import type { Project } from "@/models/Project";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { expectWriteControls } from "@/test/gates";
import { asReader, settle } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));

const memory: Memory = {
  id: "mem-1",
  workspace_id: "ws-1",
  project_id: "p-1",
  kind: "",
  title: "Naming rules",
  when_to_use: "",
  body: "",
  always_included: false,
  footer: false,
  version: 1,
  created_by: "u-1",
  created_at: "",
  updated_by: "u-1",
  updated_at: "2026-09-16T12:00:00Z",
};

const renderPane = (showLocked: boolean) => {
  mocks.get.mockImplementation(asReader({ showLocked, permissions: ["memories:read"] }));
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <MemoriesListPane memories={[memory]} project={{ id: "p-1", name: "Web", prefix: "WEB" } as Project} selectedId={undefined} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "p-1" });
});

describe.each([
  { who: "a reader", showLocked: false },
  { who: "a reader whose role shows locked actions", showLocked: true },
])("the memories list for $who", ({ showLocked }) => {
  it("hides or locks New memory and the always-included switch", async () => {
    renderPane(showLocked);
    await screen.findByText("Naming rules");
    await expectWriteControls(
      [
        { role: "button", name: "New memory", reason: "Your role can't create memories" },
        { role: "switch", name: "Always include Naming rules", reason: "Your role can't change memories" },
      ],
      showLocked,
    );
  });

  it("hides or locks a row's menu items", async () => {
    renderPane(showLocked);
    await settle();
    await userEvent.setup().pointer({ keys: "[MouseRight]", target: await screen.findByText("Naming rules") });

    for (const [name, reason] of [
      ["Move to", "Your role can't move memories"],
      ["Clone", "Your role can't clone memories to another project"],
      ["Delete", "Your role can't delete memories"],
    ] as const) {
      if (!showLocked) expect(screen.queryByRole("menuitem", { name: new RegExp(name) })).not.toBeInTheDocument();
      if (showLocked) {
        const item = await screen.findByRole("menuitem", { name: new RegExp(name) });
        expect(item).toHaveAttribute("aria-disabled", "true");
        expect(item).toHaveAccessibleDescription(reason);
      }
    }
  });
});
