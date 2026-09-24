import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { BoardCreateMenu } from "@/components/board/BoardCreateMenu";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const mockReferenceData = () =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Books", prefix: "BKS" }] };
    if (url === "/api/ticket-types") {
      return {
        data: [
          { id: "tt-task", name: "task", body_template: "" },
          { id: "tt-bug", name: "bug", body_template: "## Steps to reproduce\n\n" },
        ],
      };
    }
    if (url === "/api/tickets") return { data: [{ id: "t-1", project_id: "p-1", number: 1, title: "books page" }] };
    return { data: [] };
  });

const renderWithDialogs = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <BoardCreateMenu onNewTicket={() => {}} onNewCategory={() => {}} />
      <ContextAwareConfirmation.ConfirmationRoot />
    </QueryClientProvider>,
  );

describe("BoardCreateMenu", () => {
  it("fires onNewTicket from the menu and closes it", async () => {
    const user = userEvent.setup();
    const onNewTicket = vi.fn();
    render(<BoardCreateMenu onNewTicket={onNewTicket} onNewCategory={() => {}} />);

    await user.click(screen.getByRole("button", { name: "Add" }));
    await user.click(await screen.findByRole("button", { name: "New ticket" }));

    expect(onNewTicket).toHaveBeenCalledOnce();
    expect(screen.queryByRole("button", { name: "New ticket" })).not.toBeInTheDocument();
  });

  it("fires onNewCategory from the menu", async () => {
    const user = userEvent.setup();
    const onNewCategory = vi.fn();
    render(<BoardCreateMenu onNewTicket={() => {}} onNewCategory={onNewCategory} />);

    await user.click(screen.getByRole("button", { name: "Add" }));
    await user.click(await screen.findByRole("button", { name: "New category" }));

    expect(onNewCategory).toHaveBeenCalledOnce();
  });

  it("reports a bug from the board, refusing it until an origin is picked or marked unknown", async () => {
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-9" } });
    const user = userEvent.setup();
    renderWithDialogs();

    await user.click(screen.getByRole("button", { name: "Add" }));
    await user.click(await screen.findByRole("button", { name: "Report a bug" }));
    const dialog = await screen.findByRole("dialog");
    await vi.waitFor(() => expect(within(dialog).getByRole("textbox", { name: "Body" })).toHaveValue("## Steps to reproduce\n\n"));
    await user.type(within(dialog).getByRole("textbox", { name: "Title" }), "random crash");
    await user.click(within(dialog).getByRole("button", { name: "Report bug" }));
    expect(await within(dialog).findByText("Pick the ticket this bug was found in, or tick Origin unknown")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole("checkbox", { name: "Origin unknown" }));
    await user.click(within(dialog).getByRole("button", { name: "Report bug" }));
    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        "/api/tickets",
        expect.objectContaining({ title: "random crash", type_id: "tt-bug", origin_unknown: true, origin_id: "" }),
      ),
    );
  });

  it("files a board bug against a picked origin", async () => {
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-9" } });
    const user = userEvent.setup();
    renderWithDialogs();

    await user.click(screen.getByRole("button", { name: "Add" }));
    await user.click(await screen.findByRole("button", { name: "Report a bug" }));
    const dialog = await screen.findByRole("dialog");
    await user.type(await within(dialog).findByRole("textbox", { name: "Title" }), "books 500s");
    await user.click(within(dialog).getByRole("button", { name: "Found in…" }));
    await user.click(await screen.findByRole("button", { name: /BKS-1/ }));
    expect(within(dialog).getByRole("button", { name: "Found in BKS-1" })).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Report bug" }));
    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/tickets", expect.objectContaining({ origin_id: "t-1", origin_unknown: false })),
    );
  });
});
