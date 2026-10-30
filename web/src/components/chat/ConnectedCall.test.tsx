import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Room } from "livekit-client";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConnectedCall } from "@/components/chat/ConnectedCall";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

vi.mock("@livekit/components-react", () => ({
  RoomContext: { Provider: ({ children }: { children: React.ReactNode }) => children },
  useParticipants: () => [
    { identity: "u1", name: "Lena Kowalski" },
    { identity: "u2", name: "Sam Okafor" },
  ],
}));
vi.mock("@/components/voice/VoiceStage", () => ({ VoiceStage: () => <div data-testid="stage" /> }));
vi.mock("@/hooks/PeopleHooks", () => ({
  usePersonLookup: () => (identity: string) => ({ user_id: identity, login: identity, display_name: "", avatar_url: "" }),
}));

const initialState = useVoiceCallStore.getState();

beforeEach(() => {
  useVoiceCallStore.setState(initialState, true);
});

describe("ConnectedCall", () => {
  it("hides the stage behind one row with who is in the call, and Show call brings it back", async () => {
    const user = userEvent.setup();
    render(<ConnectedCall room={{} as Room} />);
    expect(screen.getByTestId("stage")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Hide call" }));
    expect(screen.queryByTestId("stage")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("2 in the call");
    expect(screen.getByRole("button", { name: "Leave call" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show call" }));
    expect(screen.getByTestId("stage")).toBeInTheDocument();
  });

  it("shows the stage again for the next call", () => {
    useVoiceCallStore.setState({ stageCollapsed: true });
    useVoiceCallStore.getState().leave();

    expect(useVoiceCallStore.getState().stageCollapsed).toBe(false);
  });
});
