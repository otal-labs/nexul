import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, it, vi } from "vitest";

import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { InvitationRow } from "@/components/member/InvitationRow";
import { TeamAccountActions } from "@/components/team/TeamAccountActions";
import type { ActiveInvitation } from "@/models/Invitation";
import type { TeamPerson } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { expectWriteControls, type WriteControl } from "@/test/gates";
import { asReader } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));

const invitation: ActiveInvitation = {
  id: "inv-1",
  invited_by: "u-1",
  created_at: "2026-10-01T00:00:00Z",
  expires_at: "2026-10-20T00:00:00Z",
  grants: [{ workspace_id: "ws-1", workspace_name: "Acme", role_id: "r-1", role_name: "Editor", roles: [{ id: "r-1", name: "Editor" }] }],
};

const person = (status: TeamPerson["status"]): TeamPerson => ({
  id: "u-bob",
  login: "bob",
  name: "Bob",
  avatar_url: "",
  status,
  created_at: "",
  providers: [],
  online: false,
  last_seen_at: null,
  workspaces: [],
});

const reason = (what: string) => `Your role can't ${what}`;

// What a reader of the team is offered: inviting, ending a link, and each account action.
const surfaces: { name: string; ui: React.ReactElement; controls: WriteControl[] }[] = [
  { name: "inviting", ui: <CreateInvitationDialog onCreated={() => {}} />, controls: [{ role: "button", name: "Invite", reason: reason("invite people") }] },
  { name: "a link", ui: <InvitationRow invitation={invitation} inviter="Alice" onRevoke={() => {}} disabled={false} />, controls: [{ role: "button", name: "Revoke invitation", reason: reason("revoke invitations") }] },
  {
    name: "an active account",
    ui: <TeamAccountActions person={person("active")} />,
    controls: [
      { role: "button", name: "Disable", reason: reason("disable accounts") },
      { role: "button", name: "Remove account", reason: reason("remove accounts") },
    ],
  },
  { name: "a disabled account", ui: <TeamAccountActions person={person("disabled")} />, controls: [{ role: "button", name: "Reactivate", reason: reason("reactivate accounts") }] },
  { name: "a removed account", ui: <TeamAccountActions person={person("removed")} />, controls: [{ role: "button", name: "Restore", reason: reason("restore accounts") }] },
];

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
});

describe.each([
  { who: "a reader", showLocked: false },
  { who: "a reader whose role shows locked actions", showLocked: true },
])("the team for $who", ({ showLocked }) => {
  it.each(surfaces)("$name", async ({ ui, controls }) => {
    mocks.get.mockImplementation(asReader({ showLocked, permissions: ["members:read"] }));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <ContextAwareConfirmation.ConfirmationRoot />
        {ui}
      </QueryClientProvider>,
    );
    await expectWriteControls(controls, showLocked);
  });
});
