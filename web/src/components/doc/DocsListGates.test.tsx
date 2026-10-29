import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocsListPane } from "@/components/doc/DocsListPane";
import type { DocListItem } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";
import type { Project } from "@/models/Project";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { expectWriteControls } from "@/test/gates";
import { asReader, settle } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));

const folder = (id: string, name: string, isDefault: boolean): DocFolder => ({ id, project_id: "p-1", name, is_default: isDefault, created_at: "", updated_at: "" });
const folders = [folder("f-main", "Main", true), folder("f-ref", "Reference", false)];

const doc: DocListItem = {
  id: "d-1",
  project_id: "p-1",
  folder_id: "f-ref",
  title: "Runbook",
  version: 1,
  archived: false,
  locked: false,
  can_open: true,
  created_at: "2026-01-01T09:00:00Z",
  updated_at: "2026-01-01T09:00:00Z",
};

const rest = (url: string) => (url === "/api/docs/folders" ? folders : []);

const renderPane = (showLocked: boolean) => {
  mocks.get.mockImplementation(asReader({ showLocked, permissions: ["docs:read"] }, rest));
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <DocsListPane docs={[doc]} project={{ id: "p-1", name: "Web", prefix: "WEB" } as Project} selectedId={undefined} search="" onSearch={() => undefined} />
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
])("the docs list for $who", ({ showLocked }) => {
  it("hides or locks the buttons that create", async () => {
    renderPane(showLocked);
    await screen.findByText("Runbook");
    await expectWriteControls(
      [
        { role: "button", name: "New doc", reason: "Your role can't create docs" },
        { role: "button", name: "New folder", reason: "Your role can't create folders" },
        { role: "button", name: "New doc in Reference", reason: "Your role can't create docs" },
      ],
      showLocked,
    );
  });

  it("hides or locks a row's menu items", async () => {
    renderPane(showLocked);
    await settle();
    await userEvent.setup().pointer({ keys: "[MouseRight]", target: await screen.findByText("Runbook") });

    for (const [name, reason] of [
      ["Lock", "Your role can't lock docs"],
      ["Archive", "Your role can't archive docs"],
      ["Move to folder", "Your role can't move docs"],
      ["Clone", "Your role can't clone docs into another project"],
      ["Delete", "Your role can't delete docs"],
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
