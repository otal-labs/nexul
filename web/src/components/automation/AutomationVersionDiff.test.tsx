import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { AutomationVersionStatus } from "@/enums/Automation";
import type { AutomationVersion } from "@/models/AutomationVersion";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);
vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: () => "error",
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const version = (sequence: number, status: AutomationVersionStatus): AutomationVersion => ({
  id: `v${sequence}`,
  automation_id: "a1",
  sequence,
  code: "",
  pusher_id: "u1",
  message: `Change ${sequence}`,
  status,
  created_at: "2026-08-01T00:00:00Z",
});

const bodies: Record<string, string> = {
  "/api/automations/a1/versions/1": "const a = 1;\nreturn true;",
  "/api/automations/a1/versions/2": "const a = 1;\nreturn false;",
};

const renderDiff = (active: AutomationVersion | null) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AutomationVersionDiff automationId="a1" pending={version(2, AutomationVersionStatus.Pending)} active={active} />
    </QueryClientProvider>,
  );
};

describe("AutomationVersionDiff", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.get.mockImplementation(async (url: string) => ({ data: { version: Number(url.split("/").at(-1)), body: bodies[url] } }));
  });

  it("diffs the pending version against the active one", async () => {
    renderDiff(version(1, AutomationVersionStatus.Active));

    expect(await screen.findByText("true;")).toBeInTheDocument();
    expect(screen.getByText("false;")).toBeInTheDocument();
    expect(screen.getByText("Change 2")).toBeInTheDocument();
  });

  it("shows the whole pending file when nothing is active yet", async () => {
    renderDiff(null);

    expect(await screen.findByText("return false;")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledTimes(1);
  });

  it("shows the whole pending file when only it kept a source, rather than diff a source against a bundle", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <AutomationVersionDiff automationId="a1" pending={{ ...version(2, AutomationVersionStatus.Pending), source: "return false;" }} active={version(1, AutomationVersionStatus.Active)} />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("return false;")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledTimes(1);
  });

  it("merges the pending version", async () => {
    mocks.post.mockResolvedValue({ data: version(2, AutomationVersionStatus.Active) });
    const user = userEvent.setup();
    renderDiff(version(1, AutomationVersionStatus.Active));

    await user.click(screen.getByRole("button", { name: "Merge #2" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automations/a1/versions/v2/merge");
  });
});
