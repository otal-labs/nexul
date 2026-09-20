import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationVersionRow } from "@/components/automation/AutomationVersionRow";
import { AutomationVersionStatus } from "@/enums/Automation";
import type { AutomationVersion } from "@/models/AutomationVersion";

const mocks = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const version = (overrides: Partial<AutomationVersion> = {}): AutomationVersion => ({
  id: "v1",
  automation_id: "a1",
  sequence: 3,
  code: "return true;",
  pusher_id: "nexul-seed",
  message: "Upgrade default code",
  status: AutomationVersionStatus.Inactive,
  created_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const renderRow = (v: AutomationVersion, canUpdate: boolean) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ul>
        <AutomationVersionRow automationId="a1" version={v} canUpdate={canUpdate} />
      </ul>
    </QueryClientProvider>,
  );
};

describe("AutomationVersionRow", () => {
  beforeEach(() => {
    mocks.post.mockReset();
  });

  it("shows the sequence, status, message, and labels the seed pusher as Nexul", () => {
    renderRow(version(), true);
    expect(screen.getByText("#3")).toBeInTheDocument();
    expect(screen.getByText("Upgrade default code")).toBeInTheDocument();
    expect(screen.getByText(/Nexul/)).toBeInTheDocument();
  });

  it("hides rollback for the active version", () => {
    renderRow(version({ status: AutomationVersionStatus.Active }), true);
    expect(screen.queryByRole("button", { name: /Rollback/ })).not.toBeInTheDocument();
  });

  it("hides rollback when the caller lacks automations:write", () => {
    renderRow(version(), false);
    expect(screen.queryByRole("button", { name: /Rollback/ })).not.toBeInTheDocument();
  });

  it("rolls back after confirming", async () => {
    mocks.post.mockResolvedValue({ data: version({ status: AutomationVersionStatus.Active }) });
    const user = userEvent.setup();
    renderRow(version(), true);

    await user.click(screen.getByRole("button", { name: /Rollback/ }));
    await user.click(await screen.findByRole("button", { name: "Confirm" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automations/a1/versions/v1/rollback");
  });
});
