import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TicketTypeRow } from "@/components/settings/TicketTypeRow";
import type { TicketType } from "@/models/TicketType";

const mocks = vi.hoisted(() => ({ put: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: mocks.put, post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const bug: TicketType = {
  id: "tt-bug",
  name: "bug",
  position: 1,
  color: "",
  body_template: "## Steps to reproduce\n\n",
  created_at: "",
  updated_at: "",
};

const renderRow = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ul>
        <TicketTypeRow type={bug} />
      </ul>
    </QueryClientProvider>,
  );

describe("TicketTypeRow template", () => {
  beforeEach(() => {
    mocks.put.mockReset();
    mocks.errorMessage.mockReset();
  });

  it("opens the type's current template and saves the edit", async () => {
    mocks.put.mockResolvedValue({ data: { ...bug, body_template: "## Steps\n" } });
    const user = userEvent.setup();
    renderRow();

    await user.click(screen.getByRole("button", { name: "Actions for bug" }));
    await user.click(screen.getByRole("button", { name: "Edit template" }));
    const textarea = await screen.findByLabelText("Template (markdown)");
    expect(textarea).toHaveValue("## Steps to reproduce\n\n");

    await user.clear(textarea);
    await user.type(textarea, "## Steps{Enter}");
    await user.click(screen.getByRole("button", { name: "Save template" }));

    await vi.waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/api/ticket-types/tt-bug/template", { body_template: "## Steps\n" }),
    );
  });

  it("keeps the dialog open when saving fails", async () => {
    mocks.put.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Save failed");
    const user = userEvent.setup();
    renderRow();

    await user.click(screen.getByRole("button", { name: "Actions for bug" }));
    await user.click(screen.getByRole("button", { name: "Edit template" }));
    await screen.findByLabelText("Template (markdown)");
    await user.click(screen.getByRole("button", { name: "Save template" }));

    await vi.waitFor(() => expect(mocks.put).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});
