import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutoPlaysList } from "@/components/autoplay/AutoPlaysList";
import type { AutoPlay } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { expectWriteControls } from "@/test/gates";
import { asReader } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));

const play = { id: "pl-1", workspace_id: "ws-1", label: "Fix with AI", type: "ticket" } as Play;

const autoPlay: AutoPlay = {
  id: "ap-1",
  play_id: "pl-1",
  workspace_id: "ws-1",
  enabled: true,
  moment: "ticket.unblocked",
  moment_stage: null,
  conditions: { match: "all", groups: [] },
  priority: { rules: [], otherwise: "normal" },
  once_within_minutes: 1440,
  run_on: "developer",
  run_as: "",
  computer_ids: [],
  computer_id: "",
  memory_ids: [],
  created_by: "u1",
  created_at: "",
  updated_at: "",
};

const rest = (url: string) => {
  if (url === "/api/workspaces/ws-1/plays/pl-1/auto-plays") return [autoPlay];
  if (url === "/api/workspaces/ws-1/plays/auto-play-limits") return { daily_cap_per_ticket: 5 };
  return [];
};

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
});

describe.each([
  { who: "a reader", showLocked: false },
  { who: "a reader whose role shows locked actions", showLocked: true },
])("a play's auto plays for $who", ({ showLocked }) => {
  it("hides or locks Add auto play, the switch, and the row's menu", async () => {
    mocks.get.mockImplementation(asReader({ showLocked, permissions: ["autoplays:read"] }, rest));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <AutoPlaysList play={play} onOpen={() => {}} />
      </QueryClientProvider>,
    );
    await screen.findByRole("button", { name: /becomes unblocked/ });
    await expectWriteControls(
      [
        { role: "button", name: "Add auto play", reason: "Your role can't add auto plays" },
        { role: "switch", name: "Switch auto play ap-1 on or off", reason: "Your role can't switch auto plays on or off" },
      ],
      showLocked,
    );

    await userEvent.setup().click(screen.getByRole("button", { name: "Actions for auto play ap-1" }));
    expect(await screen.findByRole("button", { name: "View" })).toBeInTheDocument();
    for (const [name, what] of [
      ["Duplicate", "duplicate auto plays"],
      ["Delete", "delete auto plays"],
    ] as const) {
      if (!showLocked) expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
      if (showLocked) {
        const item = await screen.findByRole("button", { name });
        expect(item).toHaveAttribute("aria-disabled", "true");
        expect(item).toHaveAccessibleDescription(`Your role can't ${what}`);
      }
    }
  });
});
