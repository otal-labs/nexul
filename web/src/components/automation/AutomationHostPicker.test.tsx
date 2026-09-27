import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationHostPicker } from "@/components/automation/AutomationHostPicker";
import { AutomationKind } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, patch: mocks.patch }, errorMessage: () => "error" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const hostFields = { machine: "prod", os: "linux", arch: "amd64", version: "v1", connected: true, last_seen: "2026-08-01T00:00:00Z" };
const hosts = [
  { id: "h-instance", name: "instance", ...hostFields },
  { id: "h-jobs", name: "jobs-1", ...hostFields },
];

const automation = (hostId: string | null): Automation => ({
  id: "a1",
  name: "Ticket finished",
  description: "",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: [],
  config_schema: {},
  config_values: {},
  scopes: [],
  host_id: hostId,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
});

const renderPicker = (hostId: string | null) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <AutomationHostPicker automation={automation(hostId)} />
    </QueryClientProvider>,
  );

describe("AutomationHostPicker", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.get.mockResolvedValue({ data: hosts });
    mocks.patch.mockResolvedValue({ data: automation("h-jobs") });
  });

  it("shows an unplaced automation on the built-in host, listing the instance host once", async () => {
    const user = userEvent.setup();
    renderPicker(null);

    const picker = screen.getByRole("combobox", { name: "Runs on" });
    expect(picker).toHaveTextContent("instance (built in)");
    await user.click(picker);
    expect(await screen.findAllByRole("option")).toHaveLength(2);
  });

  it("moves the automation to another host", async () => {
    const user = userEvent.setup();
    renderPicker(null);

    await pickOption(user, "Runs on", "jobs-1");

    await waitFor(() => expect(mocks.patch).toHaveBeenCalledWith("/api/automations/a1/host", { host_id: "h-jobs" }));
  });

  it("moves it back to the built-in host as no host", async () => {
    const user = userEvent.setup();
    renderPicker("h-jobs");

    await pickOption(user, "Runs on", "instance (built in)");

    await waitFor(() => expect(mocks.patch).toHaveBeenCalledWith("/api/automations/a1/host", { host_id: null }));
  });
});
