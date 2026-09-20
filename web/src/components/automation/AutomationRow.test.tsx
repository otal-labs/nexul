import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationRow } from "@/components/automation/AutomationRow";
import { AutomationKind } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";

const mocks = vi.hoisted(() => ({ patch: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { patch: mocks.patch },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const automation = (overrides: Partial<Automation> = {}): Automation => ({
  id: "a1",
  name: "Ticket finished",
  description: "Moves a ticket to done once every linked PR is merged",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: [],
  config_schema: { properties: { targetStatus: { type: "string", format: "status", title: "Target status" } }, required: ["targetStatus"] },
  config_values: {},
  scopes: ["tickets:write"],
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const renderRow = (a: Automation) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AutomationRow automation={a} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("AutomationRow", () => {
  beforeEach(() => {
    mocks.patch.mockReset();
  });

  it("shows name, description, kind badge, and the needs-configuration badge when required config is missing", () => {
    renderRow(automation());

    expect(screen.getByText("Ticket finished")).toBeInTheDocument();
    expect(screen.getByText(/Moves a ticket to done/)).toBeInTheDocument();
    expect(screen.getByText("Default")).toBeInTheDocument();
    expect(screen.getByText("Needs configuration")).toBeInTheDocument();
  });

  it("does not show needs-configuration once required config is filled in", () => {
    renderRow(automation({ config_values: { targetStatus: "status-1" } }));

    expect(screen.queryByText("Needs configuration")).not.toBeInTheDocument();
  });

  it("toggles enabled via PATCH when the switch is flipped", async () => {
    mocks.patch.mockResolvedValue({ data: automation({ enabled: false }) });
    const user = userEvent.setup();
    renderRow(automation({ enabled: true }));

    await user.click(screen.getByRole("switch"));

    expect(mocks.patch).toHaveBeenCalledWith("/api/automations/a1/enabled", { enabled: false });
  });
});
