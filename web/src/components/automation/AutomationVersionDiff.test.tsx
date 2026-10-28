import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { AutomationVersionStatus } from "@/enums/Automation";
import type { AutomationVersionDiff as VersionDiffData } from "@/models/AutomationVersion";

const mocks = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);
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

const renderDiff = (diff: VersionDiffData) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AutomationVersionDiff automationId="a1" diff={diff} />
    </QueryClientProvider>,
  );
};

describe("AutomationVersionDiff", () => {
  beforeEach(() => {
    mocks.post.mockReset();
  });

  it("renders removed and added lines for the pending vs active code", () => {
    renderDiff(diffData());

    expect(screen.getByText("return true;")).toBeInTheDocument();
    expect(screen.getByText("return false;")).toBeInTheDocument();
  });

  it("compares the sources when both versions kept one, so the inlined SDK stays out of the diff", () => {
    const { active, pending } = diffData();
    renderDiff({ active: { ...active!, source: "log(1)" }, pending: { ...pending!, source: "log(2)" } });

    expect(screen.getByText("log(1)")).toBeInTheDocument();
    expect(screen.getByText("log(2)")).toBeInTheDocument();
    expect(screen.queryByText("return true;")).not.toBeInTheDocument();
  });

  it("falls back to the bundles, and says so, when the active version kept no source", () => {
    const { active, pending } = diffData();
    renderDiff({ active: active!, pending: { ...pending!, source: "log(2)" } });

    expect(screen.getByText("return false;")).toBeInTheDocument();
    expect(screen.getByText(/bundled code/)).toBeInTheDocument();
  });

  it("shows no pending version message and no merge button when nothing is pending", () => {
    renderDiff({ active: diffData().active, pending: null });
    expect(screen.getByText("No pending version")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Merge" })).not.toBeInTheDocument();
  });

  it("merges the pending version via POST on click", async () => {
    mocks.post.mockResolvedValue({ data: diffData().pending });
    const user = userEvent.setup();
    renderDiff(diffData());

    await user.click(screen.getByRole("button", { name: "Merge" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automations/a1/versions/v2/merge");
  });
});
