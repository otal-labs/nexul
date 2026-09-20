import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { BoardSettingsSection } from "@/components/settings/BoardSettingsSection";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, patch: mocks.patch, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const statusColumn = (overrides: Record<string, unknown> = {}) => ({
  id: "open",
  name: "Open",
  position: 0,
  kind: "progress",
  icon: "",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const ticketType = (overrides: Record<string, unknown> = {}) => ({
  id: "task",
  name: "task",
  position: 0,
  color: "",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const PROJECT_ID = "proj-1";

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <BoardSettingsSection projectId={PROJECT_ID} />
    </QueryClientProvider>,
  );
};

describe("BoardSettingsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/statuses") return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
  });

  it("shows an error when the status list fails to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Statuses failed");
    renderSection();
    expect(await screen.findByText("Statuses failed")).toBeInTheDocument();
  });

  it("shows empty states for both lists", async () => {
    renderSection();
    expect(await screen.findByText(/no backlog statuses/i)).toBeInTheDocument();
    expect(screen.getByText(/no done statuses/i)).toBeInTheDocument();
    expect(screen.getByText(/no ticket types yet/i)).toBeInTheDocument();
  });

  it("does not show the empty states while statuses are still loading", () => {
    mocks.get.mockReturnValue(new Promise(() => {}));
    renderSection();
    expect(screen.queryByText(/no backlog statuses/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/no ticket types yet/i)).not.toBeInTheDocument();
  });

  it("adds a status column preset to the stage's kind", async () => {
    mocks.post.mockResolvedValue({
      data: statusColumn({ id: "blocked", name: "Blocked", position: 4 }),
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Add progress status" }));
    await user.type(screen.getByLabelText("New status name"), "Blocked");
    await user.click(screen.getByRole("button", { name: "Add column" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/statuses", {
      project_id: PROJECT_ID,
      name: "Blocked",
      kind: "progress",
      icon: "",
    });
  });

  it("adds a done status column preset to that stage's kind", async () => {
    mocks.post.mockResolvedValue({
      data: statusColumn({ id: "done", name: "Done", position: 4, kind: "done" }),
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Add done status" }));
    await user.type(screen.getByLabelText("New status name"), "Done");
    await user.click(screen.getByRole("button", { name: "Add column" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/statuses", {
      project_id: PROJECT_ID,
      name: "Done",
      kind: "done",
      icon: "",
    });
  });

  it("adds a status column with a chosen icon", async () => {
    mocks.post.mockResolvedValue({
      data: statusColumn({ id: "blocked", name: "Blocked", position: 4, icon: "CircleDot" }),
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Add progress status" }));
    await user.type(await screen.findByLabelText("New status name"), "Blocked");
    const iconPicker = within(screen.getByRole("radiogroup", { name: "New status icon" }));
    await user.click(iconPicker.getByRole("radio", { name: "CircleDot" }));
    await user.click(screen.getByRole("button", { name: "Add column" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/statuses", {
      project_id: PROJECT_ID,
      name: "Blocked",
      kind: "progress",
      icon: "CircleDot",
    });
  });

  it("does not call the api for an empty status name", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Add progress status" }));
    await user.click(await screen.findByRole("button", { name: "Add column" }));

    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("closes the add-status form when the group's '+' is clicked again", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Add progress status" }));
    expect(screen.getByLabelText("New status name")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add progress status" }));
    expect(screen.queryByLabelText("New status name")).not.toBeInTheDocument();
  });

  it("renders 'Add column' as an icon-only button, not visible text", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Add progress status" }));
    const addColumn = await screen.findByRole("button", { name: "Add column" });
    expect(addColumn).not.toHaveTextContent("Add column");
  });

  it("removes a status column", async () => {
    mocks.del.mockResolvedValue({ data: undefined });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/statuses") {
        return Promise.resolve({ data: [statusColumn({ id: "blocked", name: "Blocked", position: 4 })] });
      }
      return Promise.resolve({ data: [] });
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for Blocked" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    expect(mocks.del).toHaveBeenCalledWith("/api/statuses/blocked");
  });

  it("reorders status columns with the arrows", async () => {
    mocks.post.mockResolvedValue({ data: undefined });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/statuses") {
        return Promise.resolve({
          data: [
            statusColumn(),
            statusColumn({ id: "done", name: "Done", position: 1, kind: "done" }),
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for Open" }));
    await user.click(screen.getByRole("button", { name: "Move down" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/statuses/reorder", {
      project_id: PROJECT_ID,
      ids: ["done", "open"],
    });
  });

  it("disables reorder buttons at the list edges", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/statuses") {
        return Promise.resolve({
          data: [
            statusColumn(),
            statusColumn({ id: "done", name: "Done", position: 1, kind: "done" }),
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });
    renderSection();

    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Actions for Open" }));
    expect(screen.getByRole("button", { name: "Move up" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Move down" })).not.toBeDisabled();
    await user.keyboard("{Escape}");

    await user.click(screen.getByRole("button", { name: "Actions for Done" }));
    expect(screen.getByRole("button", { name: "Move up" })).not.toBeDisabled();
    expect(screen.getByRole("button", { name: "Move down" })).toBeDisabled();
  });

  it("renames a status column inline, resending its current icon", async () => {
    mocks.patch.mockResolvedValue({ data: statusColumn({ icon: "Circle" }) });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/statuses") {
        return Promise.resolve({
          data: [
            statusColumn({ icon: "Circle" }),
            statusColumn({ id: "done", name: "Done", position: 1, kind: "done" }),
          ],
        });
      }
      return Promise.resolve({ data: [] });
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for Open" }));
    await user.click(screen.getByRole("button", { name: "Rename" }));
    const input = screen.getByLabelText("Status name");
    await user.clear(input);
    await user.type(input, "In review");
    await user.keyboard("{Enter}");

    expect(mocks.patch).toHaveBeenCalledWith("/api/statuses/open", {
      name: "In review",
      kind: "progress",
      icon: "Circle",
    });
  });

  it("changes a status icon inline without clearing the name", async () => {
    mocks.patch.mockResolvedValue({ data: statusColumn({ icon: "CircleX" }) });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/statuses") return Promise.resolve({ data: [statusColumn()] });
      return Promise.resolve({ data: [] });
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for Open" }));
    await user.click(screen.getByRole("button", { name: "Rename" }));
    const iconPicker = within(screen.getByRole("radiogroup", { name: "Icon" }));
    await user.click(iconPicker.getByRole("radio", { name: "CircleX" }));
    await user.click(screen.getByText("Status columns"));

    expect(mocks.patch).toHaveBeenCalledWith("/api/statuses/open", {
      name: "Open",
      kind: "progress",
      icon: "CircleX",
    });
  });

  it("keeps the icon picker reachable when tabbing out of the name field", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/statuses") return Promise.resolve({ data: [statusColumn()] });
      return Promise.resolve({ data: [] });
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for Open" }));
    await user.click(screen.getByRole("button", { name: "Rename" }));
    await user.type(screen.getByLabelText("Status name"), " updated");
    await user.tab();

    expect(screen.getByRole("radiogroup", { name: "Icon" })).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("adds a ticket type", async () => {
    mocks.post.mockResolvedValue({ data: ticketType() });
    const user = userEvent.setup();
    renderSection();

    await user.type(await screen.findByLabelText("New ticket type"), "bug");
    await user.click(screen.getByRole("button", { name: "Add type" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/ticket-types", { project_id: PROJECT_ID, name: "bug" });
  });

  it("removes a ticket type", async () => {
    mocks.del.mockResolvedValue({ data: undefined });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/ticket-types") return Promise.resolve({ data: [ticketType()] });
      return Promise.resolve({ data: [] });
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for task" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    expect(mocks.del).toHaveBeenCalledWith("/api/ticket-types/task");
  });

  it("renames a ticket type inline", async () => {
    mocks.patch.mockResolvedValue({ data: ticketType() });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/ticket-types") return Promise.resolve({ data: [ticketType()] });
      return Promise.resolve({ data: [] });
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for task" }));
    await user.click(screen.getByRole("button", { name: "Rename" }));
    const input = screen.getByLabelText("Ticket type name");
    await user.clear(input);
    await user.type(input, "chore");
    await user.keyboard("{Enter}");

    expect(mocks.patch).toHaveBeenCalledWith("/api/ticket-types/task", { name: "chore", color: "" });
  });

  it("sets a ticket type's color without touching its name", async () => {
    mocks.patch.mockResolvedValue({ data: ticketType({ color: "cyan" }) });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/ticket-types") return Promise.resolve({ data: [ticketType()] });
      return Promise.resolve({ data: [] });
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Color for task" }));
    const colorPicker = within(screen.getByRole("radiogroup", { name: "Color for task" }));
    await user.click(colorPicker.getByRole("radio", { name: "cyan" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/ticket-types/task", { name: "task", color: "cyan" });
  });

  it("sets a label's color from the Labels list", async () => {
    mocks.put.mockResolvedValue({ data: { label: "bug", color: "orange" } });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/tickets/labels") return Promise.resolve({ data: ["bug"] });
      return Promise.resolve({ data: [] });
    });
    mocks.post.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    const colorPicker = within(await screen.findByRole("radiogroup", { name: "Color for label bug" }));
    await user.click(colorPicker.getByRole("radio", { name: "orange" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/tickets/labels/bug/color", {
      project_id: PROJECT_ID,
      color: "orange",
    });
  });
});
