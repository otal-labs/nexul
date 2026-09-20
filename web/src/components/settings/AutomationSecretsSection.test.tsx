import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationSecretsSection } from "@/components/settings/AutomationSecretsSection";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), del: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, delete: mocks.del },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AutomationSecretsSection />
    </QueryClientProvider>,
  );
};

describe("AutomationSecretsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.del.mockReset();
  });

  it("lists secret names and timestamps, never a value", async () => {
    mocks.get.mockResolvedValue({
      data: [{ name: "SLACK_WEBHOOK_URL", created_at: "2026-08-01T00:00:00Z", updated_at: "2026-08-02T00:00:00Z" }],
    });
    renderSection();

    expect(await screen.findByText("SLACK_WEBHOOK_URL")).toBeInTheDocument();
    expect(screen.queryByDisplayValue(/./)).toBeNull();
  });

  it("saves a new secret via PUT and clears the form", async () => {
    mocks.get.mockResolvedValue({ data: [] });
    mocks.put.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    await user.type(screen.getByLabelText("Name"), "API_KEY");
    await user.type(screen.getByLabelText("Value"), "shh");
    await user.click(screen.getByRole("button", { name: "Save secret" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/automation-secrets/API_KEY", { value: "shh" });
  });

  it("rejects a name that doesn't match the allowed pattern", async () => {
    mocks.get.mockResolvedValue({ data: [] });
    const user = userEvent.setup();
    renderSection();

    await user.type(screen.getByLabelText("Name"), "1bad-name");
    await user.type(screen.getByLabelText("Value"), "shh");
    await user.click(screen.getByRole("button", { name: "Save secret" }));

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("deletes a secret via the arm-then-confirm destroy button", async () => {
    mocks.get.mockResolvedValue({
      data: [{ name: "API_KEY", created_at: "2026-08-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z" }],
    });
    mocks.del.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    await screen.findByText("API_KEY");
    await user.click(screen.getByRole("button", { name: "Delete" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));

    expect(mocks.del).toHaveBeenCalledWith("/api/automation-secrets/API_KEY");
  });
});
