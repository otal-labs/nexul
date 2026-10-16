import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StackAlertsSection } from "@/components/stack/StackAlertsSection";
import { DeployStrategy, type Stack } from "@/models/Stack";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn(), held: ["stacks:write"] as string[] }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, patch: mocks.patch }, errorMessage: () => "boom" }));

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "instance",
  strategy: DeployStrategy.Compose,
  env: { LOG_LEVEL: "info" },
  managed: true,
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
};

const SWITCH = "Open a ticket when a service crashes";

// The actions wait for the viewer's permissions, so a test settles every read before asserting what is locked.
const renderSection = async (value: Stack = stack) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <StackAlertsSection stack={value} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await vi.waitFor(() => expect(client.isFetching()).toBe(0));
  await act(() => Promise.resolve());
};

describe("StackAlertsSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.held = ["stacks:write"];
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/permissions/explain") return { data: { effective: [] } };
      return { data: { role_name: "Member", permissions: [], projects: [{ project_id: "proj-1", actions: mocks.held }] } };
    });
    mocks.patch.mockResolvedValue({ data: stack });
  });

  it("starts off, explains what it does, and saves the whole stack with the switch on", async () => {
    const user = userEvent.setup();
    await renderSection();

    const toggle = screen.getByRole("switch", { name: SWITCH });
    expect(toggle).not.toBeChecked();
    expect(screen.getByText(/One ticket per service in the project's first stage/)).toBeInTheDocument();

    await user.click(toggle);

    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", { ...stack, crash_tickets: true });
  });

  it("switches off a stack that has it on", async () => {
    const user = userEvent.setup();
    await renderSection({ ...stack, crash_tickets: true });

    await user.click(screen.getByRole("switch", { name: SWITCH }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", { ...stack, crash_tickets: false });
  });

  it.each([
    ["an instance stack with no project", { project_id: "" }, ["stacks:write"], "Needs a project"],
    ["a reader whose variables were withheld", { env: { LOG_LEVEL: "" }, env_masked: true }, ["stacks:write"], "Your role can't see this stack's variables"],
  ])("is disabled for %s", async (_name, change, held, reason) => {
    mocks.held = held;
    const user = userEvent.setup();
    await renderSection({ ...stack, ...change });

    const toggle = screen.getByRole("switch", { name: SWITCH });
    expect(toggle).toBeDisabled();
    expect(screen.getByText(reason)).toBeInTheDocument();
    await user.click(toggle);
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("shows the state but does not flip for someone without stacks:write", async () => {
    mocks.held = [];
    const user = userEvent.setup();
    await renderSection({ ...stack, crash_tickets: true });

    const toggle = screen.getByRole("switch", { name: SWITCH });
    expect(toggle).toBeChecked();
    await user.click(toggle);
    expect(mocks.patch).not.toHaveBeenCalled();
  });
});
