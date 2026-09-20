import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationTokenSection } from "@/components/automation/AutomationTokenSection";
import { AutomationKind } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), del: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, delete: mocks.del },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const automation = (overrides: Partial<Automation> = {}): Automation => ({
  id: "a1",
  name: "Ticket finished",
  description: "",
  kind: AutomationKind.Custom,
  enabled: true,
  subscriptions: [],
  config_schema: {},
  config_values: {},
  scopes: ["tickets:write", "board:read"],
  token_prefix: "ab12",
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const renderSection = (a: Automation) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AutomationTokenSection automation={a} />
    </QueryClientProvider>,
  );
};

describe("AutomationTokenSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.del.mockReset();
    mocks.get.mockResolvedValue({ data: { instance_url: "https://demo.nexul.com" } });
  });

  it("shows the granted scopes and token prefix", () => {
    renderSection(automation());
    expect(screen.getByText("tickets:write")).toBeInTheDocument();
    expect(screen.getByText("board:read")).toBeInTheDocument();
    expect(screen.getByText("dep_…ab12")).toBeInTheDocument();
  });

  it("reveals a fresh token after rotating", async () => {
    mocks.post.mockResolvedValue({ data: { automation: automation(), token: "dep_new_token" } });
    const user = userEvent.setup();
    renderSection(automation());

    await user.click(screen.getByRole("button", { name: "Rotate" }));

    expect(await screen.findByText("dep_new_token")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/automations/a1/token");
  });

  it("revokes the token via arm-then-confirm", async () => {
    mocks.del.mockResolvedValue({ data: automation({ token_revoked_at: "2026-08-03T00:00:00Z" }) });
    const user = userEvent.setup();
    renderSection(automation());

    await user.click(screen.getByRole("button", { name: "Revoke" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));

    expect(mocks.del).toHaveBeenCalledWith("/api/automations/a1/token");
  });
});
