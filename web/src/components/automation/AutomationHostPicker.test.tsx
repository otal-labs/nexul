import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationHostPicker } from "@/components/automation/AutomationHostPicker";
import { AutomationKind } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn() }));
const card = (name: string, actions: string[]) => ({
  id: `m-${name}`, name, stack_root: "/data/nexul", first_seen: "2026-08-01T00:00:00Z", last_seen: "2026-08-12T12:00:00Z", actions,
});

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);
vi.mock("@/api/client", () => ({ api: { get: mocks.get, patch: mocks.patch }, errorMessage: () => "error" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const hostFields = { machine: "prod", os: "linux", arch: "amd64", version: "v1", connected: true, last_seen: "2026-08-01T00:00:00Z" };
const hosts = [
  { id: "h-instance", name: "instance", ...hostFields },
  { id: "h-jobs", name: "jobs-1", ...hostFields },
  { id: "h-seen", name: "seen-1", ...hostFields, machine: "seen-only" },
  { id: "h-new", name: "fresh-1", ...hostFields, machine: "not-in-nexul" },
];
const machines = [card("prod", ["machines:see", "machines:deploy"]), card("seen-only", ["machines:see"])];

const automation = (hostId: string | null): Automation => ({
  id: "a1",
  workspace_id: "ws-1",
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
    mocks.get.mockImplementation(async (url: string) => ({ data: url === "/api/machines" ? machines : hosts }));
    mocks.patch.mockResolvedValue({ data: automation("h-jobs") });
  });

  it("lists the instance host and the hosts on machines the viewer may deploy to, or that Nexul doesn't know", async () => {
    const user = userEvent.setup();
    renderPicker(null);

    const picker = screen.getByRole("combobox", { name: "Runs on" });
    expect(picker).toHaveTextContent("instance (built in)");
    await user.click(picker);
    expect((await screen.findAllByRole("option")).map((option) => option.textContent)).toEqual(["instance (built in)", "jobs-1", "fresh-1"]);
  });

  it("keeps the host an automation already runs on named but not pickable when Deploy to there is gone", async () => {
    const user = userEvent.setup();
    renderPicker("h-seen");

    await user.click(screen.getByRole("combobox", { name: "Runs on" }));

    expect(await screen.findByRole("option", { name: "seen-1" })).toHaveAttribute("aria-disabled", "true");
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
