import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { InstanceAccessSection } from "@/components/settings/InstanceAccessSection";
import type { PermissionAnswer, PermissionGrant, PermissionSource } from "@/models/Permission";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const catalog = [
  { value: "dns:read", label: "Read DNS", domain: "dns", action: "read", area: "instance" },
  { value: "dns:write", label: "Change DNS", domain: "dns", action: "write", area: "instance" },
  { value: "dns:delete", label: "Delete DNS", domain: "dns", action: "delete", area: "instance" },
];
const dnsActions = ["dns:read", "dns:write", "dns:delete"];
const areas = [
  { key: "instance", label: "the instance", actions: ["instance:read", "instance:write", "events:read"] },
  { key: "dns", label: "DNS and gateways", actions: dnsActions },
  { key: "connector:github", label: "connector GitHub", actions: ["connectors:read", "connectors:write"] },
];

const team = {
  people: [
    { id: "bob", login: "bob", name: "", avatar_url: "", status: "active", workspaces: [] },
    { id: "sam", login: "sam", name: "", avatar_url: "", status: "removed", workspaces: [] },
  ],
  workspaces: [
    { id: "ws-1", name: "Acme", can_manage_members: true, roles: [{ id: "r-owner", name: "Owner", is_owner: true }, { id: "r-dev", name: "Developer", is_owner: false }] },
    { id: "ws-2", name: "Globex", can_manage_members: true, roles: [{ id: "r-ops", name: "Ops", is_owner: false }] },
  ],
  can_manage_accounts: true,
};

const source = (over: Partial<PermissionSource>): PermissionSource => ({
  kind: "role", entity_type: "workspace", entity_id: "ws-1", target_type: "role", target_id: "r-dev", state: "allow",
  entity_name: "Acme", target_name: "Developer", sentence: "Allowed by role Developer", ...over,
});
const verdict = (allowed: boolean, over: Partial<PermissionSource>) => ({ allowed, source: source(over) });
const owner = verdict(true, { kind: "owner", target_name: "Owner", sentence: "Owner of Acme" });
const byDeveloper = verdict(true, {});
const notGranted = verdict(false, { state: "deny", sentence: "Not allowed: role Developer doesn't grant it" });
const bobAllowed = verdict(true, { kind: "rule", entity_type: "instance_area", entity_id: "dns", target_type: "person", target_id: "bob", entity_name: "DNS and gateways", target_name: "bob", sentence: "Allowed on DNS and gateways for bob" });
const answer = (action: string, now: typeof owner, fallback = now): PermissionAnswer => ({ action, ...now, fallback });
const explain = (effective: PermissionAnswer[]) => ({ data: { resource_type: "instance_area", resource_id: "dns", user_id: "", effective } });

let instanceOwner = true;
let grants: PermissionGrant[] = [];
// bob holds Read DNS through Developer in Acme and nothing more, until a rule names him.
let bobAnswers: PermissionAnswer[] = [];

const mockGet = (url: string, config?: { params?: Record<string, string> }) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: { id: "u-owner" }, instance_permissions: [], instance_owner: instanceOwner } });
  if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog, instance_areas: areas } });
  if (url === "/api/permissions?resource_type=instance_area&resource_id=dns") return Promise.resolve({ data: { grants } });
  if (url === "/api/team") return Promise.resolve({ data: team });
  if (url === "/api/permissions/explain" && config?.params?.user_id === "bob") return Promise.resolve(explain(bobAnswers));
  if (url === "/api/permissions/explain") return Promise.resolve(explain(dnsActions.map((a) => answer(a, owner))));
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

let user = userEvent.setup();

const renderSection = (section = "dns") => {
  user = userEvent.setup();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <InstanceAccessSection section={section} />
    </QueryClientProvider>,
  );
};

const openDnsPanel = async () => {
  renderSection();
  await user.click(await screen.findByRole("button", { name: /DNS and gateways/ }));
  return screen.findByRole("dialog");
};

const addBob = async () => {
  await user.click(await screen.findByRole("button", { name: "Add role or person" }));
  await user.click(await screen.findByRole("button", { name: /bob/ }));
  await screen.findByRole("heading", { name: "bob" });
};

const control = (label: string) => within(screen.getByRole("radiogroup", { name: label }));

afterEach(async () => {
  if (screen.queryByRole("dialog")) await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
});

beforeEach(() => {
  instanceOwner = true;
  grants = [];
  bobAnswers = dnsActions.map((a) => answer(a, a === "dns:read" ? byDeveloper : notGranted));
  vi.mocked(api.get).mockReset().mockImplementation(mockGet as typeof api.get);
  vi.mocked(api.put).mockReset().mockResolvedValue({ data: null });
});

describe("InstanceAccessSection", () => {
  it("shows a viewer who is not the instance Owner no access section, and asks for no rules", async () => {
    instanceOwner = false;
    renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/auth/me"));
    expect(screen.queryByRole("heading", { name: "Access" })).not.toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/permissions?resource_type=instance_area&resource_id=dns");
  });

  it("lists the entry's own area with who its rules name", async () => {
    grants = [{ resource_type: "instance_area", resource_id: "dns", target: "role", user_id: "", role_id: "r-ops", allow: ["dns:read"], deny: [], created_at: "2026-10-01", updated_at: "" }];
    renderSection();
    expect(await screen.findByText("Rules for Ops in Globex")).toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(1);
  });

  it("keeps an area with no Settings entry of its own under Instance, and each connector under Connectors", async () => {
    vi.mocked(api.get).mockImplementation(((url: string) => (url.startsWith("/api/permissions?") ? Promise.resolve({ data: { grants: [] } }) : mockGet(url))) as typeof api.get);
    renderSection("instance");
    expect(await screen.findByRole("button", { name: /The instance/ })).toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(1);
  });

  it("gives a person DNS and no connector, naming each role's workspace in the picker", async () => {
    const dialog = await openDnsPanel();
    expect(within(dialog).getByText("DNS and gateways")).toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: "Add role or person" }));
    expect(await screen.findByRole("button", { name: /Developer in Acme/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Ops in Globex/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Owner in Acme/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /sam/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /bob/ }));

    await user.click(await control("Read DNS").findByRole("radio", { name: "Allow" }));
    await user.click(control("Change DNS").getByRole("radio", { name: "Allow" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1));
    expect(api.put).toHaveBeenCalledWith("/api/permissions", {
      resource_type: "instance_area", resource_ids: ["dns"], user_ids: ["bob"], actions: ["dns:read", "dns:write"], state: "allow",
    });
  });

  it("warns before a Deny that takes away a person's last way into the area", async () => {
    await openDnsPanel();
    await addBob();

    await user.click(control("Read DNS").getByRole("radio", { name: "Deny" }));
    const warning = await screen.findByRole("dialog", { name: "Deny Read DNS to bob?" });
    expect(within(warning).getByText(/beats every role they hold/)).toBeInTheDocument();
    await user.click(within(warning).getByRole("button", { name: "Cancel" }));
    expect(control("Read DNS").getByRole("radio", { name: "Fallback" })).toBeChecked();

    await user.click(control("Read DNS").getByRole("radio", { name: "Deny" }));
    await user.click(within(await screen.findByRole("dialog", { name: "Deny Read DNS to bob?" })).getByRole("button", { name: "Deny" }));
    await waitFor(() => expect(control("Read DNS").getByRole("radio", { name: "Deny" })).toBeChecked());

    // Denying what bob doesn't hold takes nothing away, so it asks nothing.
    await user.click(control("Delete DNS").getByRole("radio", { name: "Deny" }));
    expect(control("Delete DNS").getByRole("radio", { name: "Deny" })).toBeChecked();
    expect(screen.queryByRole("dialog", { name: "Deny Delete DNS to bob?" })).not.toBeInTheDocument();
  });

  it("checks a person's answers on the area with the source of each", async () => {
    bobAnswers = dnsActions.map((a) => answer(a, a === "dns:delete" ? notGranted : bobAllowed, a === "dns:read" ? byDeveloper : notGranted));
    await openDnsPanel();
    await user.click(screen.getByRole("radio", { name: "Check access" }));
    await user.click(await screen.findByRole("button", { name: /bob/ }));
    expect(await screen.findByText("On instance area DNS and gateways, read from every rule that reaches it")).toBeInTheDocument();
    expect(screen.getAllByText((_, el) => el?.tagName === "P" && el.textContent === "Allowed on DNS and gateways for bob")).toHaveLength(2);
    expect(screen.getAllByText((_, el) => el?.tagName === "P" && el.textContent === "Not allowed: role Developer doesn't grant it")).toHaveLength(1);
  });
});
