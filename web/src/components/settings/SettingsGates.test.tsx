import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, it, vi } from "vitest";

import { ArchivedWorkspaceRow } from "@/components/settings/ArchivedWorkspaceRow";
import { DangerZoneSection } from "@/components/settings/DangerZoneSection";
import { InstanceUrlSection } from "@/components/settings/InstanceUrlSection";
import { InstanceVersionSection } from "@/components/settings/InstanceVersionSection";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { OAuthProviderSection } from "@/components/settings/OAuthProviderSection";
import type { InstanceSettings } from "@/models/User";
import type { Workspace } from "@/models/Workspace";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { expectWriteControls, type WriteControl } from "@/test/gates";
import { asReader } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));

const workspace = { id: "ws-1", name: "Acme", slug: "acme", archived_at: "2026-10-01T00:00:00Z" } as Workspace;

const settings = { instance_url: "https://nexul.example.com", settings_version: 1, oauth_callback: "https://nexul.example.com/cb", google_oauth_callback: "https://nexul.example.com/g" } as InstanceSettings;

const upgrade = {
  version: "0.3.0",
  channel: "beta",
  latest: { version: "0.3.1", url: "https://example.com/r" },
  update_available: true,
  can_upgrade: true,
  reason: "",
  upgrade: null,
};

const template = { workspace_id: "ws-1", body: "## Tests\n", questions: [], default_body: "## Tests\n", edited: false, updated_by: "", updated_at: "" };

const rest = (url: string) => {
  if (url === "/api/instance/upgrade") return upgrade;
  if (url === "/api/memories/interview-template") return template;
  if (url === "/api/workspaces") return [{ ...workspace, archived_at: undefined }];
  return [];
};

const reason = (what: string) => `Your role can't ${what}`;

// What a reader of the instance and workspace settings is offered; each surface is the section's own write.
const surfaces: { name: string; ui: React.ReactElement; controls: WriteControl[]; instance?: string[] }[] = [
  { name: "the instance address", ui: <InstanceUrlSection settings={settings} />, controls: [{ role: "button", name: "Save", reason: reason("change the instance address") }] },
  { name: "the version", ui: <InstanceVersionSection />, controls: [{ role: "button", name: /^Upgrade to/, reason: reason("upgrade the instance") }] },
  { name: "a sign-in provider", ui: <OAuthProviderSection provider="google" settings={settings} />, controls: [{ role: "button", name: /^Enable Google sign-in/, reason: reason("change sign-in providers") }] },
  { name: "an archived workspace", ui: <ArchivedWorkspaceRow workspace={workspace} />, controls: [{ role: "button", name: "Restore", reason: reason("restore workspaces") }] },
  { name: "the danger zone", ui: <DangerZoneSection />, controls: [{ role: "button", name: "Archive workspace", reason: reason("archive workspaces") }] },
  {
    name: "the interview template",
    ui: <InterviewTemplateSection />,
    controls: [
      { role: "button", name: "Save", reason: reason("edit the interview template") },
      { role: "button", name: "Clone to…", reason: reason("clone this template") },
    ],
  },
];

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
});

describe.each([
  { who: "a reader", showLocked: false },
  { who: "a reader whose role shows locked actions", showLocked: true },
])("settings for $who", ({ showLocked }) => {
  it.each(surfaces)("$name", async ({ ui, controls }) => {
    mocks.get.mockImplementation(asReader({ showLocked, permissions: ["memories:read"] }, rest));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <MemoryRouter>{ui}</MemoryRouter>
      </QueryClientProvider>,
    );
    await expectWriteControls(controls, showLocked);
  });
});
