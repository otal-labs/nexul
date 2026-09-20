import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { AutomationVersionStatus } from "@/enums/Automation";
import type { AutomationVersionDiff as VersionDiffData } from "@/models/AutomationVersion";

const mocks = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const diffData = (): VersionDiffData => ({
  active: {
    id: "v1",
    automation_id: "a1",
    sequence: 1,
    code: "return true;",
    pusher_id: "u1",
    status: AutomationVersionStatus.Active,
    created_at: "2026-08-01T00:00:00Z",
  },
  pending: {
    id: "v2",
    automation_id: "a1",
    sequence: 2,
    code: "return false;",
    pusher_id: "u1",
    status: AutomationVersionStatus.Pending,
    created_at: "2026-08-02T00:00:00Z",
  },
});

const renderDiff = (diff: VersionDiffData, canUpdate: boolean) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AutomationVersionDiff automationId="a1" diff={diff} canUpdate={canUpdate} />
    </QueryClientProvider>,
  );
};

describe("AutomationVersionDiff", () => {
  beforeEach(() => {
    mocks.post.mockReset();
  });

  it("renders removed and added lines for the pending vs active code", () => {
    renderDiff(diffData(), true);

    expect(screen.getByText("return true;")).toBeInTheDocument();
    expect(screen.getByText("return false;")).toBeInTheDocument();
  });

  it("shows the merge button only when there's a pending version and the caller can update", () => {
    renderDiff(diffData(), true);
    expect(screen.getByRole("button", { name: "Merge" })).toBeInTheDocument();
  });

  it("hides the merge button when the caller lacks automations:write", () => {
    renderDiff(diffData(), false);
    expect(screen.queryByRole("button", { name: "Merge" })).not.toBeInTheDocument();
  });

  it("shows no pending version message and no merge button when nothing is pending", () => {
    renderDiff({ active: diffData().active, pending: null }, true);
    expect(screen.getByText("No pending version")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Merge" })).not.toBeInTheDocument();
  });

  it("merges the pending version via POST on click", async () => {
    mocks.post.mockResolvedValue({ data: diffData().pending });
    const user = userEvent.setup();
    renderDiff(diffData(), true);

    await user.click(screen.getByRole("button", { name: "Merge" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automations/a1/versions/v2/merge");
  });
});
