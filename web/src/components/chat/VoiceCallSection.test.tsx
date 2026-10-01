import { MemoryRouter } from "react-router";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { VoiceCallSection } from "@/components/chat/VoiceCallSection";
import type { Conversation } from "@/models/Chat";
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
});

describe("VoiceCallSection", () => {
  it("shows an inactive join bar when not connected, and joining marks the store's active channel", async () => {
    const user = userEvent.setup();
    const join = vi.fn();
    useVoiceCallStore.setState({ join });
    renderSection(false);

    expect(screen.getByText("Not connected to this voice channel")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Join call" }));
    expect(join).toHaveBeenCalledWith("c1", "huddle");
  });

  it("after leaving a call for being alone, says why in place of the join bar and Rejoin rejoins the channel", async () => {
    const user = userEvent.setup();
    const join = vi.fn();
    const reason = "You left huddle because you were alone in the call for 5 minutes.";
    useVoiceCallStore.setState({ join, leftAlone: { conversationId: "c1", reason } });
    renderSection(false);

    expect(screen.getByText(reason)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Join call" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Rejoin" }));
    expect(join).toHaveBeenCalledWith("c1", "huddle");
  });

  it("drops the reason once the person moves away from the channel", () => {
    useVoiceCallStore.setState({ leftAlone: { conversationId: "c1", reason: "You left huddle because you were alone in the call for 5 minutes." } });
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

    expect(screen.getByText(/needs a LiveKit connector/)).toBeInTheDocument();
    const settingsLink = screen.getByRole("link", { name: "Open Settings" });
    expect(settingsLink).toHaveAttribute("href", "/settings/connectors");
    // No disabled/dead join control in this state.
    expect(screen.queryByRole("button", { name: "Join call" })).not.toBeInTheDocument();
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

    expect(screen.getByText("network unreachable")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(retry).toHaveBeenCalled();
  });

  it("keeps the Open Settings link away from a viewer who can't open Connectors", () => {
    access.sections = [];
    stageCall("not_configured");
    renderSection(true);

    expect(screen.getByText(/needs a LiveKit connector/)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open Settings" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Dismiss" })).toBeInTheDocument();
  });
});
