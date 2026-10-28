import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
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

const conversations = [
  { id: "c-eng", workspace_id: "ws-1", kind: "channel", name: "eng", created_by: "u-1", created_at: "", updated_at: "", private: false },
  { id: "c-dm", workspace_id: "ws-1", kind: "dm", created_by: "u-1", created_at: "", updated_at: "", private: false },
];

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
      if (url === "/api/chat/conversations") return { data: conversations };
      if (url === "/api/conversations/c-eng/botwebhooks") return { data: [{ id: "bot-1", conversation_id: "c-eng", name: "Alerts", created_by: "u-1", created_at: "", updated_at: "", post_count: 0 }] };
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
    expect(within(screen.getByRole("region", { name: "Crash tickets" })).getByText(reason)).toBeInTheDocument();
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

  it("names a channel from its channels only, then a bot of it, saving the whole stack each time", async () => {
    const user = userEvent.setup();
    await renderSection();

    expect(screen.queryByRole("combobox", { name: "Post as" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("combobox", { name: "Channel" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["Off", "#eng"]);
    await user.click(screen.getByRole("option", { name: "#eng" }));
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/stacks/stack-1", { ...stack, crash_notice_conversation_id: "c-eng", crash_notice_bot_id: "" });
  });

  it("posts as a bot of the named channel, and Off clears both", async () => {
    const user = userEvent.setup();
    const aimed = { ...stack, crash_notice_conversation_id: "c-eng" };
    await renderSection(aimed);

    await user.click(screen.getByRole("combobox", { name: "Post as" }));
    await user.click(await screen.findByRole("option", { name: "Alerts" }));
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/stacks/stack-1", { ...aimed, crash_notice_bot_id: "bot-1" });

    await user.click(screen.getByRole("combobox", { name: "Channel" }));
    await user.click(await screen.findByRole("option", { name: "Off" }));
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/stacks/stack-1", { ...aimed, crash_notice_conversation_id: "", crash_notice_bot_id: "" });
  });

  it("can't name a channel for a stack outside every project", async () => {
    await renderSection({ ...stack, project_id: "" });

    expect(screen.getByRole("combobox", { name: "Channel" })).toBeDisabled();
    expect(within(screen.getByRole("region", { name: "Crash notices" })).getByText("Needs a project")).toBeInTheDocument();
  });
});
