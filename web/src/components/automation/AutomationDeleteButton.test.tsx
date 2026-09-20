import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationDeleteButton } from "@/components/automation/AutomationDeleteButton";

const mocks = vi.hoisted(() => ({ del: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { delete: mocks.del },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderButton = (onDeleted: () => void) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <AutomationDeleteButton automationId="a1" onDeleted={onDeleted} />
    </QueryClientProvider>,
  );
};

describe("AutomationDeleteButton", () => {
  beforeEach(() => {
    mocks.del.mockReset();
  });

  it("does nothing without confirming", async () => {
    const onDeleted = vi.fn();
    const user = userEvent.setup();
    renderButton(onDeleted);

    await user.click(screen.getByRole("button", { name: "Delete automation" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(mocks.del).not.toHaveBeenCalled();
    expect(onDeleted).not.toHaveBeenCalled();
  });

  it("deletes and calls onDeleted after confirming", async () => {
    mocks.del.mockResolvedValue({ data: undefined });
    const onDeleted = vi.fn();
    const user = userEvent.setup();
    renderButton(onDeleted);

    await user.click(screen.getByRole("button", { name: "Delete automation" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));

    expect(mocks.del).toHaveBeenCalledWith("/api/automations/a1");
    expect(onDeleted).toHaveBeenCalledTimes(1);
  });
});
