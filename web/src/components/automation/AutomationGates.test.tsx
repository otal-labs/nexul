import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, it, vi } from "vitest";

import { AutomationDeleteButton } from "@/components/automation/AutomationDeleteButton";
import { AutomationRow } from "@/components/automation/AutomationRow";
import { AutomationSecretsSection } from "@/components/automation/AutomationSecretsSection";
import { AutomationTokenSection } from "@/components/automation/AutomationTokenSection";
import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { AutomationVersionRow } from "@/components/automation/AutomationVersionRow";
import { NewAutomationDialog } from "@/components/automation/NewAutomationDialog";
import { AutomationHostsSection } from "@/components/automationHost/AutomationHostsSection";
import { AutomationKind, AutomationVersionStatus } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { expectWriteControls, type WriteControl } from "@/test/gates";
import { asReader } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));

const automation: Automation = {
  id: "a1",
  workspace_id: "ws-1",
  name: "Ticket finished",
  description: "",
  kind: AutomationKind.Custom,
  enabled: true,
  subscriptions: [],
  config_schema: {},
  config_values: {},
  scopes: [],
  token_prefix: "ab12",
  host_id: null,
  created_at: "",
  updated_at: "",
};

const version = { id: "v2", automation_id: "a1", sequence: 2, code: "b", pusher_id: "u1", message: "", status: AutomationVersionStatus.Pending, created_at: "" };

const rest = (url: string) => {
  if (url === "/api/automation-secrets") return [{ name: "API_KEY", created_at: "", updated_at: "" }];
  if (url === "/api/automation-hosts") return [{ id: "h1", name: "builder", machine: "m", version: "1", connected: true, last_seen: "" }];
  if (url === "/api/team") return { people: [] };
  return [];
};

const reason = (what: string) => `Your role can't ${what}`;

// What a reader is offered on every automations surface, one row per surface.
const surfaces: { name: string; ui: React.ReactElement; controls: WriteControl[] }[] = [
  { name: "a row", ui: <AutomationRow automation={automation} />, controls: [{ role: "switch", name: "Disable Ticket finished", reason: reason("switch automations on or off") }] },
  { name: "New automation", ui: <NewAutomationDialog />, controls: [{ role: "button", name: "New automation", reason: reason("create automations") }] },
  {
    name: "the token",
    ui: <AutomationTokenSection automation={automation} />,
    controls: [
      { role: "button", name: "Revoke", reason: reason("revoke automation tokens") },
      { role: "button", name: "Rotate", reason: reason("rotate automation tokens") },
    ],
  },
  {
    name: "the secrets",
    ui: <AutomationSecretsSection />,
    controls: [
      { role: "button", name: "Add secret", reason: reason("add automation secrets") },
      { role: "button", name: "Replace API_KEY", reason: reason("replace automation secrets") },
      { role: "button", name: "Delete API_KEY", reason: reason("delete automation secrets") },
    ],
  },
  {
    name: "the versions",
    ui: (
      <>
        <AutomationVersionDiff automationId="a1" diff={{ active: null, pending: version }} />
        <AutomationVersionRow automationId="a1" version={{ ...version, status: AutomationVersionStatus.Inactive }} />
      </>
    ),
    controls: [
      { role: "button", name: "Merge", reason: reason("merge automation versions") },
      { role: "button", name: /Rollback/, reason: reason("roll back automations") },
    ],
  },
  {
    name: "the hosts",
    ui: <AutomationHostsSection />,
    controls: [
      { role: "button", name: "Add automations host", reason: reason("add automations hosts") },
      { role: "button", name: "Remove builder", reason: reason("remove automations hosts") },
    ],
  },
  { name: "the danger zone", ui: <AutomationDeleteButton automationId="a1" onDeleted={() => {}} />, controls: [{ role: "button", name: "Delete automation", reason: reason("delete automations") }] },
];

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe.each([
  { who: "a reader", showLocked: false },
  { who: "a reader whose role shows locked actions", showLocked: true },
])("automations for $who", ({ showLocked }) => {
  it.each(surfaces)("$name", async ({ ui, controls }) => {
    mocks.get.mockImplementation(asReader({ showLocked, permissions: [] }, rest));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <MemoryRouter>{ui}</MemoryRouter>
      </QueryClientProvider>,
    );
    await expectWriteControls(controls, showLocked);
  });
});
