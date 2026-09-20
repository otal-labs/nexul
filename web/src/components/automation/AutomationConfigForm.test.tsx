import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationConfigForm } from "@/components/automation/AutomationConfigForm";
import { AutomationKind } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, patch: mocks.patch },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

// selectedWorkspaceId feeds the channel field's fetch; a real store import
// keeps this test honest about where that id comes from.
vi.mock("@/stores/workspaceStore", () => ({
  useWorkspaceStore: (selector: (s: { selectedWorkspaceId: string }) => unknown) =>
    selector({ selectedWorkspaceId: "w1" }),
}));

const automation = (): Automation => ({
  id: "a1",
  name: "Ticket finished",
  description: "",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: [],
  config_schema: {
    type: "object",
    properties: {
      targetStatus: { type: "string", format: "status", title: "Target status", project_id: "p1" },
      channel: { type: "string", format: "channel", title: "Notify channel" },
      note: { type: "string", title: "Note" },
    },
    required: ["targetStatus"],
  },
  config_values: {},
  scopes: [],
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
});

const renderForm = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AutomationConfigForm automation={automation()} />
    </QueryClientProvider>,
  );
};

describe("AutomationConfigForm", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.patch.mockReset();
    mocks.get.mockImplementation((url: string) => {
      if (url.includes("/statuses")) return Promise.resolve({ data: [{ id: "s1", name: "Done" }] });
      if (url.includes("/chat/conversations")) {
        return Promise.resolve({ data: [{ id: "c1", kind: "channel", name: "general" }] });
      }
      return Promise.resolve({ data: [] });
    });
  });

  it("maps status → dropdown, channel → picker, string → input", () => {
    renderForm();

    expect(screen.getByText("Target status")).toBeInTheDocument();
    expect(screen.getByText("Notify channel")).toBeInTheDocument();
    expect(screen.getByLabelText("Note")).toBeInTheDocument();
    expect(screen.getAllByRole("combobox")).toHaveLength(2);
  });

  it("submits the free-text field's value through PATCH /config", async () => {
    mocks.patch.mockResolvedValue({ data: automation() });
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText("Note"), "hello");
    await user.click(screen.getByRole("button", { name: "Save configuration" }));

    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/automations/a1/config",
      expect.objectContaining({ config_values: expect.objectContaining({ note: "hello" }) }),
    );
  });

  it("shows a no-config message when the schema has no fields", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const bare: Automation = { ...automation(), config_schema: { properties: {} } };
    render(
      <QueryClientProvider client={client}>
        <AutomationConfigForm automation={bare} />
      </QueryClientProvider>,
    );

    expect(screen.getByText("This automation has no config knobs")).toBeInTheDocument();
  });
});
