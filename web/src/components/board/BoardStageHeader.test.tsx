import { DndContext } from "@dnd-kit/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { BoardStageHeader } from "@/components/board/BoardStageHeader";
import type { BoardStatus } from "@/models/Status";
import { useBoardStore } from "@/stores/boardStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asViewer, memberWho } from "@/test/viewer";

vi.mock("@/api/client", async () => {
  const { asOwner } = await import("@/test/viewer");
  return { api: { get: vi.fn(asOwner()), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() }, errorMessage: vi.fn() };
});

const column: BoardStatus = {
  id: "open",
  name: "Open",
  position: 0,
  kind: "progress",
  icon: "",
  created_at: "",
  updated_at: "",
};

const done: BoardStatus = { ...column, id: "done", name: "Done", position: 1, kind: "done" };

const renderHeader = (props: Partial<Parameters<typeof BoardStageHeader>[0]> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DndContext>
        <BoardStageHeader columns={[column]} counts={{ open: 2 }} onAddTicket={() => {}} {...props} />
      </DndContext>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  useBoardStore.setState({ doneOpen: false });
  vi.mocked(api.get).mockImplementation(asViewer(memberWho(["tickets:write", "projects:write"], false)));
});

describe("BoardStageHeader", () => {
  it("names the stage and counts its tickets across lanes", () => {
    renderHeader();
    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.getByLabelText("2 tickets")).toHaveTextContent("2");
  });

  it("calls onAddTicket with this stage's status id when the + button is clicked", async () => {
    const user = userEvent.setup();
    const onAddTicket = vi.fn();
    renderHeader({ onAddTicket });
    await user.click(await screen.findByRole("button", { name: "New ticket in Open" }));
    expect(onAddTicket).toHaveBeenCalledWith(null, "open");
  });

  it("makes the name and count the stage's drag surface, leaving the add button outside it", async () => {
    renderHeader();
    const surface = await screen.findByRole("button", { name: "Reorder Open" });
    const add = screen.getByRole("button", { name: "New ticket in Open" });
    expect(surface).toHaveAttribute("aria-roledescription", "sortable");
    expect(surface).not.toBe(add);
    expect(surface).not.toContainElement(add);
    expect(surface).toHaveTextContent("Open");
  });

  it("renders the chosen icon instead of the dot when set", () => {
    renderHeader({ columns: [{ ...column, icon: "CircleCheckBig" }] });
    const heading = screen.getByRole("heading", { level: 4 });
    expect(heading.querySelector(".rounded-full.bg-info")).not.toBeInTheDocument();
    expect(heading.querySelector("span > svg")).toBeInTheDocument();
  });

  it("falls back to the dot for an unrecognized icon value instead of rendering nothing", () => {
    renderHeader({ columns: [{ ...column, icon: "not-a-real-icon" }] });
    const heading = screen.getByRole("heading", { level: 4 });
    expect(heading.querySelector("span > svg")).not.toBeInTheDocument();
    expect(heading.querySelector(".rounded-full.bg-info")).toBeInTheDocument();
  });

  it("folds Done by default with its count, opens on click and folds again", async () => {
    const user = userEvent.setup();
    renderHeader({ columns: [column, done], counts: { open: 2, done: 7 } });
    expect(screen.queryByRole("heading", { name: /Done/ })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show Done, 7 tickets" }));
    expect(useBoardStore.getState().doneOpen).toBe(true);
    expect(screen.getByLabelText("7 tickets")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Fold Done" }));
    expect(useBoardStore.getState().doneOpen).toBe(false);
    expect(screen.getByRole("button", { name: "Show Done, 7 tickets" })).toBeInTheDocument();
  });

  describe("for a member who may read the board and not write it", () => {
    const readOnly = (showLocked: boolean) => vi.mocked(api.get).mockImplementation(asViewer(memberWho(["tickets:read", "projects:read"], showLocked)));

    it("offers no reorder and no write control", async () => {
      readOnly(false);
      renderHeader();
      expect(await screen.findByText("Open")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Reorder Open" })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "New ticket in Open" })).not.toBeInTheDocument();
    });

    it("locks the add button and offers no way to drag the stage when a role shows locked actions", async () => {
      readOnly(true);
      const user = userEvent.setup();
      const onAddTicket = vi.fn();
      renderHeader({ onAddTicket });

      const add = await screen.findByRole("button", { name: "New ticket in Open" });
      expect(add).toHaveAttribute("aria-disabled", "true");
      expect(screen.queryByRole("button", { name: "Reorder Open" })).not.toBeInTheDocument();
      await user.click(add);
      expect(onAddTicket).not.toHaveBeenCalled();
    });
  });
});
