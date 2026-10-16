import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StackDangerZoneSection } from "@/components/stack/StackDangerZoneSection";
import { asOwner } from "@/test/viewer";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Stack } from "@/models/Stack";

const mocks = vi.hoisted(() => ({ get: vi.fn(), _delete: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, delete: mocks._delete },
  errorMessage: () => "boom",
}));

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "edge",
  slug: "edge",
  machine: "instance",
  strategy: "run",
  managed: true,
  created_at: "2026-08-12T09:00:00Z",
  updated_at: "2026-08-12T09:00:00Z",
};

const openConfirm = async (s: Stack) => {
  const user = userEvent.setup();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ContextAwareConfirmation.ConfirmationRoot />
        <StackDangerZoneSection stack={s} projectPath="/p" hostnames={[]} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await user.click(await screen.findByRole("button", { name: "Delete stack" }));
  return within(await screen.findByRole("dialog"));
};

describe("StackDangerZoneSection delete confirm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.get.mockImplementation(asOwner());
  });

  it("promises a managed stack's containers are removed by its runner", async () => {
    const dialog = await openConfirm(stack);
    expect(dialog.getByText(/runner removes its containers, once it is online if it is not now/)).toBeInTheDocument();
    await userEvent.click(dialog.getByRole("button", { name: "Cancel" }));
  });

  it("says an adopted stack's containers keep running", async () => {
    const dialog = await openConfirm({ ...stack, managed: false });
    expect(dialog.getByText(/containers keep running, because they were started outside Nexul/)).toBeInTheDocument();
    expect(dialog.queryByText(/removes its containers/)).not.toBeInTheDocument();
    await userEvent.click(dialog.getByRole("button", { name: "Cancel" }));
  });
});
