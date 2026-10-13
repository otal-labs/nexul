import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import type { PermissionAnswer, PermissionSource } from "@/models/Permission";
import type { PanelEntity } from "@/models/PermissionPanel";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const actions = ["docs:read", "docs:write", "docs:delete", "permissions:write"];
const catalog = [
  { value: "docs:read", label: "Read docs", domain: "docs", action: "read", area: "project" },
  { value: "docs:write", label: "Create and update docs", domain: "docs", action: "write", area: "project" },
  { value: "docs:delete", label: "Delete docs", domain: "docs", action: "delete", area: "project" },
];
const person = (login: string) => ({ user_id: login, login, display_name: "", avatar_url: "https://example.com/a.png" });
const people = ["alice", "bob", "lena", "sam"].map(person);

const source = (over: Partial<PermissionSource>): PermissionSource => ({
  kind: "role", entity_type: "workspace", entity_id: "ws-1", target_type: "role", target_id: "r-dev", state: "allow",
  entity_name: "Acme", target_name: "Developer", sentence: "Allowed by role Developer", ...over,
});
const byRole = { allowed: true, source: source({}) };
const answer = (action: string, verdict = byRole, fallback = byRole): PermissionAnswer => ({ action, ...verdict, fallback });
const explain = (effective: PermissionAnswer[]) => ({ data: { resource_type: "doc", resource_id: "doc-1", user_id: "", effective } });
const role = (id: string, name: string, owner = false) => ({ id, workspace_id: "ws-1", name, permissions: [], is_owner_role: owner, created_at: "", updated_at: "" });
const roles = [role("r-owner", "Owner", true), role("r-dev", "Developer"), role("r-client", "Client"), role("r-ops", "Ops")];
const notListed = { allowed: false, source: source({ kind: "not_in_project", entity_type: "project", entity_id: "p-1", target_type: "role", target_id: "r-client", state: "deny", entity_name: "Storefront", target_name: "Client", sentence: "Not listed on project Storefront" }) };

const bobDenied = { allowed: false, source: source({ kind: "rule", entity_type: "doc", entity_id: "doc-1", target_type: "person", target_id: "bob", state: "deny", entity_name: "Pricing", target_name: "bob", sentence: "Denied on doc Pricing for bob" }) };
const samOutside = { allowed: false, source: source({ kind: "not_in_project", entity_type: "project", target_type: "person", target_id: "sam", state: "deny", entity_name: "Storefront", target_name: "sam", sentence: "Not found: sam isn't in project Storefront" }) };
const explanations: Record<string, PermissionAnswer[]> = {
  "": actions.map((a) => answer(a, a === "docs:delete" ? { allowed: false, source: source({ state: "deny", sentence: "Not allowed: role Developer doesn't grant it" }) } : byRole)),
  bob: actions.map((a) => answer(a, a === "docs:write" ? bobDenied : byRole)),
  lena: actions.map((a) => answer(a)),
  alice: actions.map((a) => answer(a)),
  sam: actions.map((a) => answer(a, samOutside, samOutside)),
};

const grants = [
  { resource_type: "doc", resource_id: "doc-1", target: "person", user_id: "lena", allow: ["docs:read"], deny: [], created_at: "2026-10-02", updated_at: "" },
  { resource_type: "doc", resource_id: "doc-1", target: "person", user_id: "bob", allow: [], deny: ["docs:write"], created_at: "2026-10-01", updated_at: "" },
  { resource_type: "doc", resource_id: "doc-1", target: "role", user_id: "", role_id: "r-client", allow: [], deny: ["docs:read"], created_at: "2026-10-03", updated_at: "" },
];

const mockGet = (url: string, config?: { params?: Record<string, string> }) => {
  if (url === "/api/permissions?resource_type=doc&resource_id=doc-1") return Promise.resolve({ data: { grants } });
  if (url === "/api/permissions?resource_type=folder&resource_id=f-1") return Promise.resolve({ data: { grants: [] } });
  if (url === "/api/permissions?resource_type=stack&resource_id=st-1") return Promise.resolve({ data: { grants: [] } });
  if (url === "/api/permissions/explain" && config?.params?.role_id) return Promise.resolve(explain(actions.map((a) => answer(a, byRole, notListed))));
  if (url === "/api/permissions/explain") return Promise.resolve(explain(explanations[config?.params?.user_id ?? ""] ?? []));
  if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: roles });
  if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
  if (url === "/api/workspaces/ws-1/people") return Promise.resolve({ data: { people } });
  if (url === "/api/projects/p-1/people") return Promise.resolve({ data: { people: people.filter((p) => p.login !== "sam"), role_ids: ["r-dev", "r-client"] } });
  if (url === "/api/projects/p-1") return Promise.resolve({ data: { id: "p-1", name: "Storefront" } });
  if (url === "/api/docs/folders") return Promise.resolve({ data: [{ id: "f-1", name: "Sales" }] });
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

const pricing: PanelEntity = { type: "doc", id: "doc-1", name: "Pricing", workspaceId: "ws-1", projectId: "p-1", folderId: "f-1" };

const Harness = ({ entity }: { entity: PanelEntity }) => {
  const open = usePermissionsDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button type="button" onClick={async () => setResult((await open(entity)).success ? "saved" : "cancelled")}>
        Open
      </button>
      <p>{result}</p>
    </div>
  );
};

let user = userEvent.setup();

const openPanel = async (entity = pricing) => {
  user = userEvent.setup();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <Harness entity={entity} />
    </QueryClientProvider>,
  );
  await user.click(screen.getByRole("button", { name: "Open" }));
  await screen.findByRole("navigation", { name: "Roles and people with rules" });
  return user;
};

const control = (label: string) => within(screen.getByRole("radiogroup", { name: label }));

// react-confirm keeps a dialog left open across tests, so each ends with the dialog settled.
afterEach(async () => {
  if (screen.queryByText("pending")) await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByText("pending")).not.toBeInTheDocument());
});

beforeEach(() => {
  vi.mocked(api.get).mockReset().mockImplementation(mockGet as typeof api.get);
  vi.mocked(api.put).mockReset().mockResolvedValue({ data: null });
});

describe("PermissionsPanel", () => {
  it("names the doc and where it sits, and lists its people in the order their rules were made", async () => {
    await openPanel();
    expect(await screen.findByText("in Storefront › Sales", { exact: false })).toBeInTheDocument();
    const rows = within(screen.getByRole("navigation", { name: "Roles and people with rules" })).getAllByRole("listitem");
    expect(rows.map((r) => r.textContent)).toEqual(["Client1 rule", "bob1 rule", "lena1 rule"]);
  });

  it("opens a person's rules with their saved states and what applies under each Fallback", async () => {
    await openPanel();
    await user.click(screen.getByRole("button", { name: /bob/ }));
    expect(await screen.findByText("Person · holds Developer")).toBeInTheDocument();
    expect(control("Create and update docs").getByRole("radio", { name: "Deny" })).toBeChecked();
    expect(control("Read docs").getByRole("radio", { name: "Fallback" })).toBeChecked();
    // Three rows sit at Fallback and say what applies there; the Deny row says nothing.
    expect(screen.getAllByText((_, el) => el?.tagName === "P" && el.textContent === "Allowed by role Developer")).toHaveLength(3);
    expect(control("Manage this doc's access").getByRole("radio", { name: "Fallback" })).toBeChecked();
  });

  it("locks Allow on a permission the giver doesn't hold, and leaves Deny open", async () => {
    await openPanel();
    await user.click(screen.getByRole("button", { name: /lena/ }));
    const locked = await screen.findByRole("radio", { name: "Allow: You can't allow a permission you don't hold" });
    expect(locked).toBeDisabled();
    expect(control("Delete docs").getByRole("radio", { name: "Deny" })).toBeEnabled();
  });

  it("saves one request per person and state, and nothing else", async () => {
    await openPanel();
    await user.click(screen.getByRole("button", { name: /bob/ }));
    await user.click(await control("Create and update docs").findByRole("radio", { name: "Allow" }));
    await user.click(screen.getByRole("button", { name: /lena/ }));
    await user.click(await control("Read docs").findByRole("radio", { name: "Deny" }));
    await user.click(control("Delete docs").getByRole("radio", { name: "Deny" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("saved")).toBeInTheDocument();
    expect(api.put).toHaveBeenCalledTimes(2);
    expect(api.put).toHaveBeenCalledWith("/api/permissions", { doc_ids: ["doc-1"], user_ids: ["bob"], actions: ["docs:write"], state: "allow" });
    expect(api.put).toHaveBeenCalledWith("/api/permissions", { doc_ids: ["doc-1"], user_ids: ["lena"], actions: ["docs:read", "docs:delete"], state: "deny" });
  });

  it("removes a person by setting each of their rules to Fallback on Save", async () => {
    await openPanel();
    await user.click(screen.getByRole("button", { name: /bob/ }));
    await user.click(await screen.findByRole("button", { name: "Remove from this doc" }));
    expect(screen.getByText("Pick a role or person, or add one, to set rules on Pricing.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /bob/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/permissions", { doc_ids: ["doc-1"], user_ids: ["bob"], actions: ["docs:write"], state: "fallback" }),
    );
  });

  it("adds a member to the list, and refuses a role or person who isn't in the doc's project", async () => {
    await openPanel();
    await user.click(screen.getByRole("button", { name: "Add role or person" }));
    expect(await screen.findByText("Add sam to Storefront first")).toBeInTheDocument();
    expect(screen.getByText("Add Ops to Storefront first")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /sam/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Owner/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /alice/ }));
    expect(await screen.findByRole("heading", { name: "alice" })).toBeInTheDocument();
    expect(within(screen.getByRole("navigation", { name: "Roles and people with rules" })).getByText("alice")).toBeInTheDocument();
  });

  it("on a folder, names it in its project, refuses who isn't in the project, and saves its rules for the folder", async () => {
    await openPanel({ type: "folder", id: "f-1", name: "Sales", workspaceId: "ws-1", projectId: "p-1", folderId: "f-1" });
    expect(await screen.findByText("in Storefront", { exact: false })).toHaveTextContent("folder Sales in Storefront");
    await user.click(screen.getByRole("button", { name: "Add role or person" }));
    expect(await screen.findByText("Add sam to Storefront first")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /lena/ }));
    expect(await control("Manage this folder's access").findByRole("radio", { name: "Fallback" })).toBeChecked();
    await user.click(control("Read docs").getByRole("radio", { name: "Deny" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/permissions", { resource_type: "folder", resource_ids: ["f-1"], user_ids: ["lena"], actions: ["docs:read"], state: "deny" }),
    );
  });

  it("on a stack, names it in its project, refuses who isn't in the project, and saves its rules for the stack", async () => {
    await openPanel({ type: "stack", id: "st-1", name: "production", workspaceId: "ws-1", projectId: "p-1", folderId: "" });
    expect(await screen.findByText("in Storefront", { exact: false })).toHaveTextContent("stack production in Storefront");
    await user.click(screen.getByRole("button", { name: "Add role or person" }));
    expect(await screen.findByText("Add sam to Storefront first")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /lena/ }));
    await user.click((await control("Read docs").findAllByRole("radio", { name: "Deny" }))[0]!);
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/permissions", { resource_type: "stack", resource_ids: ["st-1"], user_ids: ["lena"], actions: ["docs:read"], state: "deny" }),
    );
  });

  it("sets a role's rules, says what applies under each Fallback without them, and saves them for the role", async () => {
    await openPanel();
    await user.click(screen.getByRole("button", { name: /Client/ }));
    expect(await screen.findByText("Role")).toBeInTheDocument();
    expect(control("Read docs").getByRole("radio", { name: "Deny" })).toBeChecked();
    expect(screen.getAllByText((_, el) => el?.tagName === "P" && el.textContent === "Not listed on project Storefront")).toHaveLength(3);
    expect(api.get).toHaveBeenCalledWith("/api/permissions/explain", { params: { resource_type: "doc", resource_id: "doc-1", role_id: "r-client" } });

    await user.click(control("Read docs").getByRole("radio", { name: "Fallback" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/permissions", { doc_ids: ["doc-1"], role_ids: ["r-client"], actions: ["docs:read"], state: "fallback" }),
    );
  });

  it("selects a role a source sentence names", async () => {
    await openPanel();
    await user.click(screen.getByRole("button", { name: /bob/ }));
    const [link] = await screen.findAllByRole("button", { name: "Developer" });
    await user.click(link!);
    expect(await screen.findByRole("heading", { name: "Developer" })).toBeInTheDocument();
  });

  it("checks a person's answers with the source of each, and a person outside the project once", async () => {
    await openPanel();
    await user.click(screen.getByRole("radio", { name: "Check access" }));
    await user.click(await screen.findByRole("button", { name: /bob/ }));
    expect(await screen.findByRole("heading", { name: "What bob can do" })).toBeInTheDocument();
    expect(screen.getAllByText((_, el) => el?.textContent === "Denied on doc Pricing for bob").length).toBeGreaterThan(0);

    await user.click(screen.getByRole("button", { name: /sam/ }));
    expect(await screen.findAllByText((_, el) => el?.tagName === "P" && el.textContent === "Not found: sam isn't in project Storefront")).toHaveLength(1);
    expect(screen.queryByRole("heading", { name: "Docs" })).not.toBeInTheDocument();
  });
});
