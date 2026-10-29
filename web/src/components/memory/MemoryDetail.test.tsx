import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MemoryDetail } from "@/components/memory/MemoryDetail";
import type { Memory } from "@/models/Memory";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asOwner, asReader, settle } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, put: mocks.put }, errorMessage: () => "error" }));

vi.mock("@/components/attachment/AttachmentsSection", () => ({
  AttachmentsSection: () => <div data-testid="attachments-section" />,
}));
vi.mock("@/components/memory/MemoryVersionsFeed", () => ({
  MemoryVersionsFeed: () => <div data-testid="versions-feed" />,
}));
vi.mock("@/components/memory/CloneMemoryDialog", () => ({
  CloneMemoryDialog: ({ open }: { open: boolean }) => (open ? <div data-testid="clone-dialog" /> : null),
}));
vi.mock("@/components/doc/RichTextEditor", () => ({
  RichTextEditor: () => <div data-testid="rich-text-editor" />,
}));

const memory: Memory = {
  id: "mem-1",
  workspace_id: "ws-1",
  project_id: "project-1",
  kind: "",
  title: "Deploy quirks",
  when_to_use: "when deploying",
  body: "body",
  always_included: false,
  footer: false,
  version: 3,
  created_by: "user-1",
  created_at: "2026-09-16T12:00:00Z",
  updated_by: "user-1",
  updated_at: "2026-09-16T12:00:00Z",
};

const renderDetail = async (overrides: Partial<React.ComponentProps<typeof MemoryDetail>> = {}, viewer = asOwner()) => {
  mocks.get.mockImplementation(viewer);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/memories/m1"]}>
        <Routes>
          <Route
            path="/memories/:memoryId/:tab?"
            element={
              <MemoryDetail
                memory={memory}
                onSave={vi.fn()}
                onDelete={vi.fn()}
                saving={false}
                {...overrides}
              />
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mocks.get.mockReset();
  mocks.put.mockReset();
  mocks.put.mockResolvedValue({ data: memory });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("MemoryDetail", () => {
  it("opens the clone dialog from the actions menu for a role with memories:clone", async () => {
    const user = userEvent.setup();
    await renderDetail();
    await user.click(screen.getByRole("button", { name: "Memory actions" }));
    expect(screen.queryByTestId("clone-dialog")).not.toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: "Clone to…" }));
    expect(await screen.findByTestId("clone-dialog")).toBeInTheDocument();
  });

  it.each([
    { name: "hides", showLocked: false },
    { name: "locks", showLocked: true },
  ])("$name Clone to… and Delete for a reader", async ({ showLocked }) => {
    const user = userEvent.setup();
    await renderDetail({}, asReader({ showLocked, permissions: ["memories:read"] }));
    await screen.findByRole("heading", { name: "Deploy quirks" });
    await settle();
    await user.click(screen.getByRole("button", { name: "Memory actions" }));

    for (const [name, reason] of [
      ["Clone to…", "Your role can't clone memories to another project"],
      ["Delete", "Your role can't delete memories"],
    ] as const) {
      if (!showLocked) expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
      if (showLocked) {
        const control = await screen.findByRole("button", { name });
        expect(control).toHaveAttribute("aria-disabled", "true");
        expect(control).toHaveAccessibleDescription(reason);
      }
    }
  });

  it("shows the read-only title, folder and when-to-use without memories:write", async () => {
    await renderDetail({}, asReader({ permissions: ["memories:read"] }));
    expect(await screen.findByRole("heading", { name: "Deploy quirks" })).toBeInTheDocument();
    expect(screen.getByText("when deploying")).toBeInTheDocument();
    expect(screen.queryByLabelText("Title")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Folder/ })).not.toBeInTheDocument();
  });

  it("shows editable fields with memories:write", async () => {
    await renderDetail();
    expect(await screen.findByLabelText("Title")).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "When to use" })).toHaveValue("when deploying");
    expect(screen.getByTestId("rich-text-editor")).toBeInTheDocument();
  });

  it("switches Always included from the record", async () => {
    const user = userEvent.setup();
    await renderDetail();
    await screen.findByLabelText("Title");
    await user.click(screen.getByRole("switch", { name: "Always included" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/memories/mem-1", expect.objectContaining({ always_included: true }));
  });

  it("moves the memory to the footer from its folder menu", async () => {
    const user = userEvent.setup();
    await renderDetail();
    await user.click(await screen.findByRole("button", { name: "Folder: Main" }));
    await user.click(await screen.findByRole("menuitemradio", { name: /Footer/ }));
    expect(mocks.put).toHaveBeenCalledWith("/api/memories/mem-1", expect.objectContaining({ footer: true }));
  });

  it("locks the interview memory on, says why, and counts it against the cap", async () => {
    await renderDetail({ memory: { ...memory, kind: "interview", always_included: true, body: "x".repeat(8001) } });
    expect(await screen.findByText(/can't be switched off/)).toBeInTheDocument();
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Folder/ })).not.toBeInTheDocument();
    expect(await screen.findByRole("alert")).toHaveTextContent("8,001 / 8,000 characters");
  });

  it("says the decisions log is not sent every turn, and why", async () => {
    await renderDetail({ memory: { ...memory, kind: "decisions_log" } });
    expect(await screen.findByText("Not sent every turn")).toBeInTheDocument();
    expect(screen.getByText(/when an agent needs the why/)).toBeInTheDocument();
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
  });
});
