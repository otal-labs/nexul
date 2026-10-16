import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StackErrorRateCard } from "@/components/stack/StackErrorRateCard";
import { DeployStrategy, type Stack, type StackRequests } from "@/models/Stack";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn(), held: ["stacks:write"] as string[], requests: {} as Partial<StackRequests> }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, patch: mocks.patch }, errorMessage: () => "boom" }));

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "instance",
  strategy: DeployStrategy.Compose,
  managed: true,
  error_rate: { enabled: false, percent: 5, minutes: 5, min_requests: 20 },
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
};

const SWITCH = "Alert when 5xx errors stay high";
const NO_HOSTNAME = "Needs a hostname on this stack";
const NO_ANALYTICS = "Your Cloudflare token can't read analytics; add Zone Analytics Read to it";

const renderCard = async (value: Stack = stack) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <StackErrorRateCard stack={value} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await vi.waitFor(() => expect(client.isFetching()).toBe(0));
  await act(() => Promise.resolve());
};

describe("StackErrorRateCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.held = ["stacks:write"];
    mocks.requests = { counted: true, from: "2026-10-16T09:43:00Z", requests: [], errors_5xx: [], error_rate_unavailable: "" };
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-1/requests") return { data: mocks.requests };
      if (url === "/api/permissions/explain") return { data: { effective: [] } };
      return { data: { role_name: "Member", permissions: [], projects: [{ project_id: "proj-1", actions: mocks.held }] } };
    });
    mocks.patch.mockResolvedValue({ data: stack });
  });

  it("states the rule and how it ends, and switches it on by saving the whole stack", async () => {
    const user = userEvent.setup();
    await renderCard();

    expect(screen.getByRole("heading", { name: "Error rate" })).toBeInTheDocument();
    expect(screen.getByText(/Trips on 5xx above 5% for 5 minutes, with at least 20 requests\./)).toBeInTheDocument();
    expect(screen.getByText("Recovers after 5 minutes at or under 5%.")).toBeInTheDocument();
    const toggle = screen.getByRole("switch", { name: SWITCH });
    expect(toggle).not.toBeChecked();

    await user.click(toggle);

    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", { ...stack, error_rate: { ...stack.error_rate, enabled: true } });
  });

  it("switches a rule that is on off", async () => {
    const user = userEvent.setup();
    await renderCard({ ...stack, error_rate: { ...stack.error_rate!, enabled: true } });

    await user.click(screen.getByRole("switch", { name: SWITCH }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", { ...stack, error_rate: { ...stack.error_rate, enabled: false } });
  });

  it("saves the three numbers with the rule's switch left as it is", async () => {
    const user = userEvent.setup();
    await renderCard({ ...stack, error_rate: { ...stack.error_rate!, enabled: true } });

    const percent = screen.getByLabelText("5xx above (%)");
    await user.clear(percent);
    await user.type(percent, "2.5");
    const minutes = screen.getByLabelText("For (minutes)");
    await user.clear(minutes);
    await user.type(minutes, "10");
    const floor = screen.getByLabelText("With at least (requests)");
    await user.clear(floor);
    await user.type(floor, "50");
    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save" }));

    await vi.waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", { ...stack, error_rate: { enabled: true, percent: 2.5, minutes: 10, min_requests: 50 } }),
    );
  });

  it("refuses numbers outside the bounds without saving", async () => {
    const user = userEvent.setup();
    await renderCard();

    const minutes = screen.getByLabelText("For (minutes)");
    await user.clear(minutes);
    await user.type(minutes, "90");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("Enter at most 60 minutes")).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it.each([
    ["has no hostname", NO_HOSTNAME],
    ["is a tunnel stack whose Cloudflare token can't read analytics", NO_ANALYTICS],
  ])("greys the switch out with the reason when the stack %s", async (_name, reason) => {
    mocks.requests = { ...mocks.requests, counted: false, error_rate_unavailable: reason };
    const user = userEvent.setup();
    await renderCard();

    const toggle = await screen.findByRole("switch", { name: SWITCH });
    await vi.waitFor(() => expect(toggle).toBeDisabled());
    expect(screen.getByText(reason)).toBeInTheDocument();
    await user.click(toggle);
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("still lets a rule that is already on be switched off when the stack lost its hostname", async () => {
    mocks.requests = { ...mocks.requests, counted: false, error_rate_unavailable: NO_HOSTNAME };
    const user = userEvent.setup();
    await renderCard({ ...stack, error_rate: { ...stack.error_rate!, enabled: true } });

    const toggle = screen.getByRole("switch", { name: SWITCH });
    expect(toggle).toBeEnabled();
    await user.click(toggle);
    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", { ...stack, error_rate: { ...stack.error_rate, enabled: false } });
  });

  it("shows the state but does not flip for someone without stacks:write", async () => {
    mocks.held = [];
    const user = userEvent.setup();
    await renderCard({ ...stack, error_rate: { ...stack.error_rate!, enabled: true } });

    const toggle = screen.getByRole("switch", { name: SWITCH });
    expect(toggle).toBeChecked();
    await user.click(toggle);
    expect(mocks.patch).not.toHaveBeenCalled();
    expect(screen.getByLabelText("5xx above (%)")).toBeDisabled();
  });

  it("is disabled for a reader whose variables were withheld", async () => {
    await renderCard({ ...stack, env: { LOG_LEVEL: "" }, env_masked: true });
    expect(screen.getByRole("switch", { name: SWITCH })).toBeDisabled();
    expect(screen.getByText("Your role can't see this stack's variables")).toBeInTheDocument();
  });
});
