import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MachinePeopleRow } from "@/components/runner/MachinePeopleRow";
import type { PermissionAnswer, PermissionGrant, PermissionSource } from "@/models/Permission";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const machineActions = ["machines:see", "machines:deploy", "machines:write", "machines:delete"];
const catalog = [
  { value: "machines:see", label: "See", domain: "machines", action: "see", area: "machine" },
  { value: "machines:deploy", label: "Deploy to", domain: "machines", action: "deploy", area: "machine" },
  { value: "machines:write", label: "Manage", domain: "machines", action: "write", area: "machine" },
  { value: "machines:delete", label: "Delete", domain: "machines", action: "delete", area: "machine" },
];

const team = {
  people: [{ id: "bob", login: "bob", name: "", avatar_url: "", status: "active", workspaces: [] }],
  workspaces: [
    { id: "ws-1", name: "Acme", can_manage_members: true, roles: [{ id: "r-owner", name: "Owner", is_owner: true }] },
    { id: "ws-2", name: "Globex", can_manage_members: true, roles: [{ id: "r-ops", name: "Ops", is_owner: false }] },
  ],
  can_manage_accounts: true,
};

const owner: PermissionSource = {
  kind: "owner", entity_type: "workspace", entity_id: "ws-1", target_type: "role", target_id: "r-owner", state: "allow",
  entity_name: "Acme", target_name: "Owner", sentence: "Owner of Acme",
};
const answer = (action: string): PermissionAnswer => ({ action, allowed: true, source: owner, fallback: { allowed: true, source: owner } });

let instanceOwner = true;
let grants: PermissionGrant[] = [];

const mockGet = (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: { id: "u-owner" }, instance_permissions: [], instance_owner: instanceOwner } });
  if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog, instance_areas: [] } });
  if (url === "/api/permissions?resource_type=machine&resource_id=m-1") return Promise.resolve({ data: { grants } });
  if (url === "/api/team") return Promise.resolve({ data: team });
  if (url === "/api/permissions/explain") {
    return Promise.resolve({ data: { resource_type: "machine", resource_id: "m-1", user_id: "", effective: machineActions.map(answer) } });
  }
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

let user = userEvent.setup();

const renderRow = () => {
  user = userEvent.setup();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MachinePeopleRow machine={{ id: "m-1", name: "edge-box" }} />
    </QueryClientProvider>,
  );
};

const control = (label: string) => within(screen.getByRole("radiogroup", { name: label }));

afterEach(async () => {
  if (screen.queryByRole("dialog")) await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
});

beforeEach(() => {
  instanceOwner = true;
  grants = [];
  vi.mocked(api.get).mockReset().mockImplementation(mockGet as typeof api.get);
  vi.mocked(api.put).mockReset().mockResolvedValue({ data: null });
});

describe("MachinePeopleRow", () => {
  it("says only the instance Owner is on a machine no rule names, and offers to add people", async () => {
    renderRow();
    expect(await screen.findByText("Only you")).toBeInTheDocument();
    expect(screen.getByText("Add people")).toBeInTheDocument();
  });

  it("names who its rules reach instead, and opens the panel on the machine", async () => {
    grants = [
      { resource_type: "machine", resource_id: "m-1", target: "role", user_id: "", role_id: "r-ops", allow: ["machines:see"], deny: [], created_at: "2026-10-01", updated_at: "" },
      { resource_type: "machine", resource_id: "m-1", target: "person", user_id: "bob", allow: [], deny: [], created_at: "2026-10-02", updated_at: "" },
    ];
    renderRow();
    expect(await screen.findByText("Ops in Globex")).toBeInTheDocument();
    expect(screen.queryByText("Only you")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /People on this machine/ }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("edge-box")).toBeInTheDocument();
    expect(await within(dialog).findByRole("button", { name: /Ops in Globex/ })).toBeInTheDocument();
  });

  it("gives a role of another workspace See and a person Deploy to, Manage and Delete, one write per target", async () => {
    renderRow();
    await user.click(await screen.findByRole("button", { name: /People on this machine/ }));

    await user.click(await screen.findByRole("button", { name: "Add role or person" }));
    expect(screen.queryByRole("button", { name: /Owner in Acme/ })).not.toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: /Ops in Globex/ }));
    await user.click(await control("See this machine").findByRole("radio", { name: "Allow" }));

    await user.click(screen.getByRole("button", { name: "Add role or person" }));
    await user.click(await screen.findByRole("button", { name: /bob/ }));
    await screen.findByRole("heading", { name: "bob" });
    for (const label of ["Deploy to this machine", "Manage this machine", "Delete this machine"]) {
      await user.click(control(label).getByRole("radio", { name: "Allow" }));
    }
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(api.put).toHaveBeenCalledTimes(2));
    expect(api.put).toHaveBeenCalledWith("/api/permissions", {
      resource_type: "machine", resource_ids: ["m-1"], role_ids: ["r-ops"], actions: ["machines:see"], state: "allow",
    });
    expect(api.put).toHaveBeenCalledWith("/api/permissions", {
      resource_type: "machine", resource_ids: ["m-1"], user_ids: ["bob"], actions: ["machines:deploy", "machines:write", "machines:delete"], state: "allow",
    });
  });
});
