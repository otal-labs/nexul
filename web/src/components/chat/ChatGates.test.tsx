import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Conversation } from "@nexul/client-core/chat";

import { BotsSection } from "@/components/botwebhook/BotsSection";
import { ChannelSettingsCard } from "@/components/chat/ChannelSettingsCard";
import { ChatSidebarSection } from "@/components/sidebar/ChatSidebarSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { expectWriteControls, type WriteControl } from "@/test/gates";
import { asReader, settle } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));

const channel: Conversation = { id: "c-eng", workspace_id: "ws-1", kind: "channel", name: "eng", created_by: "u1", created_at: "", updated_at: "" };

const rest = (url: string) => (url === "/api/chat/conversations" ? [channel] : []);

const reason = (what: string) => `Your role can't ${what}`;

// What a reader is offered around channels and bots; they hold only the reads, so every create, rename and delete is theirs to be refused.
const surfaces: { name: string; ui: React.ReactElement; controls: WriteControl[] }[] = [
  {
    name: "the sidebar",
    ui: <ChatSidebarSection collapsed={false} />,
    controls: [
      { role: "button", name: "New channel", reason: reason("create channels") },
      { role: "button", name: "New voice channel", reason: reason("create channels") },
      { role: "button", name: "New direct message", reason: reason("start direct messages") },
    ],
  },
  {
    name: "a channel's settings",
    ui: <ChannelSettingsCard channel={channel} onOpenPermissions={() => {}} />,
    controls: [
      { role: "button", name: "Permissions…", reason: reason("set who sees and posts in channels") },
      { role: "switch", name: "Private channel", reason: reason("change who sees channels"), rowHidden: true },
    ],
  },
  { name: "a channel's bots", ui: <BotsSection conversation={channel} />, controls: [{ role: "button", name: "New bot", reason: reason("create bots") }] },
];

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
});

describe.each([
  { who: "a reader", showLocked: false },
  { who: "a reader whose role shows locked actions", showLocked: true },
])("chat for $who", ({ showLocked }) => {
  it.each(surfaces)("$name", async ({ ui, controls }) => {
    mocks.get.mockImplementation(asReader({ showLocked, permissions: ["channels:read", "botwebhook:read"] }, rest));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>{ui}</MemoryRouter>
      </QueryClientProvider>,
    );
    await expectWriteControls(controls, showLocked);
  });

  it("a channel row's Rename and Delete", async () => {
    mocks.get.mockImplementation(asReader({ showLocked, permissions: ["channels:read"] }, rest));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ChatSidebarSection collapsed={false} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await settle();
    await userEvent.setup().pointer({ keys: "[MouseRight]", target: await screen.findByText("eng") });

    for (const [name, what] of [
      ["Rename", "rename channels"],
      ["Delete", "delete channels"],
    ] as const) {
      if (!showLocked) expect(screen.queryByRole("menuitem", { name })).not.toBeInTheDocument();
      if (showLocked) {
        const item = await screen.findByRole("menuitem", { name: new RegExp(name) });
        expect(item).toHaveAttribute("aria-disabled", "true");
        expect(item).toHaveAccessibleDescription(`Your role can't ${what}`);
      }
    }
  });
});
