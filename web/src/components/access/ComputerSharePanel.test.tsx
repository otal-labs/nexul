import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import type { PermissionAnswer } from "@/models/Permission";
import { computerEntity } from "@/models/PermissionPanel";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const actions = ["computers:see", "computers:run_agents", "computers:run_commands"];
const person = (login: string) => ({ user_id: login, login, display_name: "", avatar_url: "" });
const verdict = (allowed: boolean, kind: string, sentence: string) => ({
  allowed,
  source: { kind, entity_type: "computer", entity_id: "c-1", target_type: "person", target_id: "dave", state: allowed ? "allow" : "deny", entity_name: "", target_name: "dave", sentence },
});
const answers = (held: string[]): PermissionAnswer[] =>
  actions.map((action) => ({
    action,
    ...(held.includes(action) ? verdict(true, "rule", "Shared with dave by alice") : verdict(false, "not_shared", "Not shared with dave")),
    fallback: verdict(false, "not_shared", "Not shared with dave"),
  }));

const mockGet = (url: string, config?: { params?: Record<string, string> }) => {
  if (url === "/api/permissions?resource_type=computer&resource_id=c-1")
    return Promise.resolve({ data: { grants: [{ resource_type: "computer", resource_id: "c-1", target: "person", user_id: "dave", allow: ["computers:see"], deny: [], created_at: "2026-10-12", updated_at: "" }] } });
  if (url === "/api/permissions/explain" && config?.params?.user_id === "dave") return Promise.resolve({ data: { effective: answers(["computers:see"]) } });
  if (url === "/api/permissions/explain") return Promise.resolve({ data: { effective: actions.map((action) => ({ action, ...verdict(true, "owner", "Owner of this computer"), fallback: verdict(true, "owner", "Owner of this computer") })) } });
  if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: [] } });
  if (url === "/api/workspaces") return Promise.resolve({ data: [{ id: "ws-1" }, { id: "ws-2" }] });
  if (url === "/api/workspaces/ws-1/people") return Promise.resolve({ data: { people: [person("alice"), person("dave")] } });
  if (url === "/api/workspaces/ws-2/people") return Promise.resolve({ data: { people: [person("dave"), person("erin")] } });
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: { id: "alice" } } });
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

const Harness = () => {
  const open = usePermissionsDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button type="button" onClick={async () => setResult((await open(computerEntity({ id: "c-1", name: "Laptop" }))).success ? "saved" : "cancelled")}>
        Open
      </button>
      <p>{result}</p>
    </div>
  );
};

let user = userEvent.setup();

const openPanel = async () => {
  user = userEvent.setup();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <Harness />
    </QueryClientProvider>,
  );
  await user.click(screen.getByRole("button", { name: "Open" }));
  await screen.findByRole("navigation", { name: "Roles and people with rules" });
};

const control = (label: string) => within(screen.getByRole("radiogroup", { name: label }));

afterEach(async () => {
  if (screen.queryByText("pending")) await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByText("pending")).not.toBeInTheDocument());
});

beforeEach(() => {
  vi.mocked(api.get).mockReset().mockImplementation(mockGet as typeof api.get);
  vi.mocked(api.put).mockReset().mockResolvedValue({ data: null });
});

describe("sharing a computer", () => {
  it("warns what sharing hands over and offers each person around the owner once, no role and not the owner", async () => {
    await openPanel();
    expect(screen.getByText("Agents and commands run as your own user on this computer.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add a person" }));
    expect(await screen.findByRole("button", { name: /erin/ })).toBeInTheDocument();
    expect(screen.queryByText("Roles")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /alice/ })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("sets each permission to Allow or Fallback, Run bringing See along, and saves them for the person", async () => {
    await openPanel();
    await user.click(screen.getByRole("button", { name: /dave/ }));
    expect(await control("See this computer").findByRole("radio", { name: "Allow" })).toBeChecked();
    expect(control("Run agents").queryByRole("radio", { name: "Deny" })).not.toBeInTheDocument();
    expect(screen.getAllByText((_, el) => el?.tagName === "P" && el.textContent === "Not shared with dave")).toHaveLength(2);

    await user.click(control("See this computer").getByRole("radio", { name: "Fallback" }));
    await user.click(control("Run commands").getByRole("radio", { name: "Allow" }));
    expect(control("See this computer").getByRole("radio", { name: "Allow" })).toBeChecked();
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("saved")).toBeInTheDocument();
    expect(api.put).toHaveBeenCalledWith("/api/permissions", { resource_type: "computer", resource_ids: ["c-1"], user_ids: ["dave"], actions: ["computers:run_commands"], state: "allow" });
    expect(api.put).toHaveBeenCalledTimes(1);
  });
});
