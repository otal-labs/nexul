import { MemoryRouter } from "react-router";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Conversation } from "@nexul/client-core/chat";

import { VoiceCallSection } from "@/components/chat/VoiceCallSection";
import { useVoiceCallStore, type VoiceCallStatus } from "@/stores/voiceCallStore";

const access = vi.hoisted(() => ({ sections: ["connectors"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useCanOpenSection: (section: string) => access.sections.includes(section) }));

beforeEach(() => {
  access.sections = ["connectors"];
});

// The tile avatar lookup fetches the workspace's people via react-query; it is not what these tests
// exercise, so it's stubbed to an unknown person instead of wiring a QueryClient.
vi.mock("@/hooks/PeopleHooks", () => ({
  usePersonLookup: () => (identity: string) => ({ user_id: identity, login: identity, display_name: "", avatar_url: "" }),
}));

const room = vi.hoisted(() => ({
  occupants: [] as { identity: string; name: string }[],
  speaking: new Set<string>(),
  listens: [] as [string, boolean][],
}));
vi.mock("@/hooks/VoiceHooks", () => ({
  useVoiceOccupancy: () => ({ c1: room.occupants }),
  useVoiceLobbySpeakers: (conversationId: string, listening: boolean) => {
    room.listens.push([conversationId, listening]);
    return room.speaking;
  },
}));

const conversation: Conversation = {
  id: "c1",
  workspace_id: "ws-1",
  kind: "voice_channel",
  name: "huddle",
  created_by: "u1",
  created_at: "",
  updated_at: "",
};

// The section reads the shared call from voiceCallStore (whose actions own the real WebRTC room,
// mocked out because jsdom has no WebRTC), so tests stage state by writing the store directly.
const stageCall = (status: VoiceCallStatus, overrides?: Partial<ReturnType<typeof useVoiceCallStore.getState>>) => {
  useVoiceCallStore.setState({ activeConversationId: "c1", status, ...overrides });
};

const renderSection = (active: boolean) =>
  render(
    <MemoryRouter>
      <VoiceCallSection conversation={conversation} active={active} />
    </MemoryRouter>,
  );

const initialState = useVoiceCallStore.getState();

beforeEach(() => {
  useVoiceCallStore.setState(initialState, true);
  room.occupants = [];
  room.speaking = new Set();
  room.listens = [];
});

describe("VoiceCallSection", () => {
  it("names who is in the room before joining, and Join joins this channel", async () => {
    const user = userEvent.setup();
    const join = vi.fn();
    useVoiceCallStore.setState({ join });
    room.occupants = [
      { identity: "u2", name: "Lena Kowalski" },
      { identity: "u3", name: "Sam Okafor" },
    ];
    renderSection(false);

    expect(screen.getByText("Lena and Sam")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("2 in the call");
    await user.click(screen.getByRole("button", { name: "Join" }));
    expect(join).toHaveBeenCalledWith("c1", "huddle");
  });

  it("listens for speakers only on the join prompt of a room with people in it", () => {
    room.occupants = [{ identity: "u2", name: "Lena Kowalski" }];
    renderSection(false);
    expect(room.listens.at(-1)).toEqual(["c1", true]);
  });

  it.each([
    ["an empty room", false, []],
    ["a call being joined", true, [{ identity: "u2", name: "Lena Kowalski" }]],
  ])("does not listen in %s", (_name, active, occupants) => {
    room.occupants = occupants;
    stageCall("connecting");
    renderSection(active);
    expect(room.listens.at(-1)).toEqual(["c1", false]);
  });

  it("offers to start the call in an empty room", async () => {
    const user = userEvent.setup();
    const join = vi.fn();
    useVoiceCallStore.setState({ join });
    renderSection(false);

    expect(screen.getByText("Nobody's here")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Start call" }));
    expect(join).toHaveBeenCalledWith("c1", "huddle");
  });

  it("says which call joining would leave when another channel's call is on", () => {
    useVoiceCallStore.setState({ activeConversationId: "c2", activeChannelName: "#standup", status: "connected" });
    room.occupants = [{ identity: "u2", name: "Lena Kowalski" }];
    renderSection(false);

    expect(screen.getByRole("status")).toHaveTextContent("1 in the call · You're in #standup");
    expect(screen.getByRole("button", { name: "Switch call" })).toBeInTheDocument();
  });

  it("after leaving a call for being alone, says why in place of the join bar and Rejoin rejoins the channel", async () => {
    const user = userEvent.setup();
    const join = vi.fn();
    const reason = "You left huddle after 5 minutes alone in the call.";
    useVoiceCallStore.setState({ join, leftAlone: { conversationId: "c1", reason } });
    renderSection(false);

    expect(screen.getByText("You left after 5 minutes alone")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start call" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Rejoin" }));
    expect(join).toHaveBeenCalledWith("c1", "huddle");
  });

  it("drops the reason once the person moves away from the channel", () => {
    useVoiceCallStore.setState({ leftAlone: { conversationId: "c1", reason: "You left huddle after 5 minutes alone in the call." } });
    const { unmount } = renderSection(false);

    unmount();

    expect(useVoiceCallStore.getState().leftAlone).toBeNull();
  });

  it("shows a connecting state with a cancel affordance", () => {
    stageCall("connecting");
    renderSection(true);

    expect(screen.getByText("Connecting to huddle…")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("renders the owner-setup-needed state inline, never a dead button, and links to settings", () => {
    stageCall("not_configured");
    renderSection(true);

    expect(screen.getByText("Voice isn't set up")).toBeInTheDocument();
    const settingsLink = screen.getByRole("link", { name: "Open connectors" });
    expect(settingsLink).toHaveAttribute("href", "/settings/connectors");
    // No disabled/dead join control in this state.
    expect(screen.queryByRole("button", { name: "Start call" })).not.toBeInTheDocument();
  });

  it("dismissing the owner-setup-needed state clears the active channel", async () => {
    const user = userEvent.setup();
    stageCall("not_configured");
    renderSection(true);

    await user.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(useVoiceCallStore.getState().activeConversationId).toBeNull();
  });

  it("shows a retryable error state", async () => {
    const user = userEvent.setup();
    const retry = vi.fn();
    stageCall("error", { error: "network unreachable", retry });
    renderSection(true);

    expect(screen.getByText("Couldn't connect")).toBeInTheDocument();
    expect(screen.getByText("network unreachable")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(retry).toHaveBeenCalled();
  });

  it("tells a viewer who can't open Connectors who can fix it, instead of linking there", () => {
    access.sections = [];
    stageCall("not_configured");
    renderSection(true);

    expect(screen.getByText("Ask someone who manages connectors to add LiveKit")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open connectors" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Dismiss" })).toBeInTheDocument();
  });
});
