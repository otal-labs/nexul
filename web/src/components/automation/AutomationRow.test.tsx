import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationRow } from "@/components/automation/AutomationRow";
import { AutomationKind, AutomationRunOutcome } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";
import { automationListItems } from "@/utils/AutomationListUtility";

const mocks = vi.hoisted(() => ({ patch: vi.fn() }));

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);
vi.mock("@/api/client", () => ({
  api: { patch: mocks.patch },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const automation = (overrides: Partial<Automation> = {}): Automation => ({
  id: "a1",
  workspace_id: "ws-1",
  name: "Ticket finished",
  description: "Moves a ticket to done once every linked PR is merged",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: [],
  config_schema: { properties: { targetStatus: { type: "string", format: "status", title: "Target status" } }, required: ["targetStatus"] },
  config_values: {},
  scopes: ["tickets:write"],
  host_id: null,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const renderRow = (a: Automation) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter>
        <AutomationRow item={automationListItems([a], new Set())[0]!} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("AutomationRow", () => {
  beforeEach(() => {
    mocks.patch.mockReset();
  });

  it("shows what a healthy automation does under its name, with no chip", () => {
    renderRow(automation({ config_values: { targetStatus: "status-1" } }));

    expect(screen.getByText("Ticket finished")).toBeInTheDocument();
    expect(screen.getByText(/Moves a ticket to done/)).toBeInTheDocument();
    expect(screen.queryByText("Needs configuration")).not.toBeInTheDocument();
  });

  it("shows the needs-configuration chip while a required setting is empty", () => {
    renderRow(automation());

    expect(screen.getByText("Needs configuration")).toBeInTheDocument();
  });

  it("puts the last error in place of the description while its last run failed", () => {
    renderRow(automation({ last_run: { outcome: AutomationRunOutcome.Crash, started_at: "2026-08-01T00:00:00Z", error: "handler exceeded the 30 second limit" } }));

    expect(screen.getByText(/handler exceeded the 30 second limit/)).toBeInTheDocument();
    expect(screen.queryByText(/Moves a ticket to done/)).not.toBeInTheDocument();
  });

  it("toggles enabled via PATCH when the switch is flipped", async () => {
    mocks.patch.mockResolvedValue({ data: automation({ enabled: false }) });
    const user = userEvent.setup();
    renderRow(automation({ enabled: true }));

    await user.click(screen.getByRole("switch"));

    expect(mocks.patch).toHaveBeenCalledWith("/api/automations/a1/enabled", { enabled: false });
  });
});

describe("AutomationRow switching on with a required setting empty", () => {
  beforeEach(() => {
    mocks.patch.mockReset();
  });

  it("asks first, and a cancel leaves it off", async () => {
    const user = userEvent.setup();
    renderRow(automation({ enabled: false }));

    await user.click(screen.getByRole("switch", { name: "Enable Ticket finished" }));
    expect(await screen.findByText(/Target status is empty/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("switches on once confirmed", async () => {
    mocks.patch.mockResolvedValue({ data: automation() });
    const user = userEvent.setup();
    renderRow(automation({ enabled: false }));

    await user.click(screen.getByRole("switch", { name: "Enable Ticket finished" }));
    await user.click(await screen.findByRole("button", { name: "Switch on" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/automations/a1/enabled", { enabled: true });
  });
});
