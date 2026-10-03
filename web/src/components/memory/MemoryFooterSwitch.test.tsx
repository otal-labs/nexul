import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MemoryFooterSwitch } from "@/components/memory/MemoryFooterSwitch";
import type { Memory } from "@/models/Memory";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn() },
  errorMessage: (error: unknown) => (error as Error)?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const memory: Memory = {
  id: "mem-1",
  workspace_id: "ws-1",
  project_id: "p-1",
  kind: "",
  title: "Where tickets go next",
  when_to_use: "Concluding a run",
  body: "body",
  always_included: false,
  footer: false,
  version: 1,
  created_by: "u-1",
  created_at: "",
  updated_by: "u-1",
  updated_at: "",
};

const renderSwitch = (permissions: string[], subject: Memory = memory) => {
  vi.mocked(api.get).mockResolvedValue({ data: { role_name: "Member", permissions } });
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryFooterSwitch memory={subject} label="Footer" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.put).mockResolvedValue({ data: { ...memory, footer: true } });
});

describe("MemoryFooterSwitch", () => {
  it("saves the memory as a footer on toggle, keeping its other fields", async () => {
    const user = userEvent.setup();
    renderSwitch(["memories:write"]);

    const toggle = screen.getByRole("switch", { name: "Footer" });
    await vi.waitFor(() => expect(toggle).toBeEnabled());
    await user.click(toggle);

    expect(api.put).toHaveBeenCalledWith("/api/memories/mem-1", {
      title: "Where tickets go next",
      when_to_use: "Concluding a run",
      body: "body",
      always_included: false,
      footer: true,
    });
  });

  it("is disabled for a reader", async () => {
    renderSwitch(["memories:read"]);
    await vi.waitFor(() => expect(api.get).toHaveBeenCalled());
    expect(screen.getByRole("switch", { name: "Footer" })).toBeDisabled();
  });

  it("is fixed off for the interview memory even when the role can write", async () => {
    renderSwitch(["memories:write"], { ...memory, kind: "interview", footer: true });
    await vi.waitFor(() => expect(api.get).toHaveBeenCalled());
    const toggle = screen.getByRole("switch", { name: "Footer" });
    expect(toggle).toBeDisabled();
    expect(toggle).not.toBeChecked();
  });
});
