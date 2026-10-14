import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MachineGroup } from "@/components/runner/MachineGroup";
import type { Machine } from "@/models/Runner";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => () => false, useIsInstanceOwner: () => true }));

const machine = (actions: string[]): Machine => ({
  id: "m-1", name: "edge-box", stack_root: "/data/nexul", first_seen: "2026-08-01T00:00:00Z", last_seen: "2026-08-12T12:00:00Z", actions,
});

const held = { allowed: true, source: { kind: "role" }, fallback: { allowed: true, source: { kind: "role" } } };

beforeEach(() => {
  vi.mocked(api.get).mockReset().mockImplementation(((url: string) => {
    if (url === "/api/permissions/explain") return Promise.resolve({ data: { effective: [{ action: "machines:write", ...held }] } });
    if (url.startsWith("/api/permissions?")) return Promise.resolve({ data: { grants: [] } });
    if (url === "/api/team") return Promise.resolve({ data: { people: [], workspaces: [], can_manage_accounts: true } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  }) as typeof api.get);
});

const renderCard = (actions: string[]) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <MachineGroup machine={machine(actions)} runners={[]} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("MachineGroup people", () => {
  it("offers People on this machine, the ⋯ and a right-click Permissions… to a Manage holder", async () => {
    const user = userEvent.setup();
    renderCard(["machines:see", "machines:write"]);
    expect(await screen.findByRole("button", { name: /People on this machine/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "More actions for edge-box" })).toBeInTheDocument();

    await user.pointer({ keys: "[MouseRight]", target: screen.getByRole("heading", { name: "edge-box" }) });
    expect(await screen.findByRole("menuitem", { name: "Permissions…" })).toBeInTheDocument();
  });

  it("shows nobody else's people to someone who may only see and deploy to the machine", async () => {
    const user = userEvent.setup();
    renderCard(["machines:see", "machines:deploy"]);
    expect(await screen.findByRole("heading", { name: "edge-box" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /People on this machine/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "More actions for edge-box" })).not.toBeInTheDocument();

    await user.pointer({ keys: "[MouseRight]", target: screen.getByRole("heading", { name: "edge-box" }) });
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/permissions?resource_type=machine&resource_id=m-1");
  });
});
